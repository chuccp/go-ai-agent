package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
	"go.starlark.net/starlark"
	starlarkjson "go.starlark.net/starlarkjson"
	"go.starlark.net/syntax"
)

type ScriptNodeConfig struct {
	Script string `json:"script"`
}

func (c *ScriptNodeConfig) UnmarshalJSON(data []byte) error {
	type Alias ScriptNodeConfig
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if c.Script == "" {
		if v, ok := raw["code"].(string); ok {
			c.Script = v
		}
	}
	return nil
}

// ScriptNode Starlark (Python dialect) dynamic script node
// Predefined:
//   ctx["node_name"]["output"] — upstream node output
//   json_parse(s)               — JSON string → dict/list
//   json_string(v)              — dict/list → JSON string
//   split(s, sep)               — split by delimiter
// Assign result to the result variable
type ScriptNode struct{}

func NewScriptNode() *ScriptNode { return &ScriptNode{} }

func (n *ScriptNode) Type() string { return "script" }

func (n *ScriptNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ScriptNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Script == "" {
		return nil, fmt.Errorf("script: script is required")
	}

	ctxDict := starlark.NewDict(len(ctx.AllNodeOutputs()))
	for label, output := range ctx.AllNodeOutputs() {
		inner := starlark.NewDict(len(output.Data))
		for k, v := range output.Data {
			inner.SetKey(starlark.String(k), toStarlark(v))
		}
		ctxDict.SetKey(starlark.String(label), inner)
	}

	thread := newStarlarkThread("script")
	predeclared := starlark.StringDict{
		"ctx":    ctxDict,
		"result": starlark.None,
		"json_parse": starlark.NewBuiltin("json_parse", func(
			thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			var s starlark.String
			if err := starlark.UnpackArgs("json_parse", args, kwargs, "s", &s); err != nil {
				return nil, err
			}
			var v any
			if err := json.Unmarshal([]byte(string(s)), &v); err != nil {
				return nil, fmt.Errorf("json_parse: %w", err)
			}
			return toStarlark(v), nil
		}),
		"json_string": starlark.NewBuiltin("json_string", func(
			thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			var v starlark.Value
			if err := starlark.UnpackArgs("json_string", args, kwargs, "v", &v); err != nil {
				return nil, err
			}
			b, err := json.Marshal(fromStarlark(v))
			if err != nil {
				return nil, err
			}
			return starlark.String(string(b)), nil
		}),
		"split": starlark.NewBuiltin("split", func(
			thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple,
		) (starlark.Value, error) {
			var s, sep starlark.String
			if err := starlark.UnpackArgs("split", args, kwargs, "s", &s, "sep", &sep); err != nil {
				return nil, err
			}
			parts := strings.Split(string(s), string(sep))
			list := make([]starlark.Value, 0)
			for _, p := range parts {
				if p = strings.TrimSpace(p); p != "" {
					list = append(list, starlark.String(p))
				}
			}
			return starlark.NewList(list), nil
		}),
	}

	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "script.py", cfg.Script, predeclared)
	if err != nil {
		return nil, fmt.Errorf("script error: %w", err)
	}

	resultVal := globals["result"]
	if resultVal == nil || resultVal == starlark.None {
		return nil, fmt.Errorf("script: result variable not set, assign to 'result'")
	}

	resultStr := resultVal.String()
	isJSON := false
	switch resultVal.(type) {
	case *starlark.Dict, *starlark.List:
		isJSON = true
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: resultStr,
			"is_json": isJSON,
		},
		Status: engine.StatusSuccess,
	}, nil
}

func toStarlark(v any) starlark.Value {
	switch val := v.(type) {
	case string:
		return starlark.String(val)
	case int:
		return starlark.MakeInt64(int64(val))
	case int64:
		return starlark.MakeInt64(val)
	case float64:
		return starlark.Float(val)
	case bool:
		return starlark.Bool(val)
	case []any:
		list := make([]starlark.Value, len(val))
		for i, item := range val {
			list[i] = toStarlark(item)
		}
		return starlark.NewList(list)
	case map[string]any:
		dict := starlark.NewDict(len(val))
		for k, item := range val {
			dict.SetKey(starlark.String(k), toStarlark(item))
		}
		return dict
	default:
		return starlark.String(fmt.Sprintf("%v", v))
	}
}

func fromStarlark(v starlark.Value) any {
	switch val := v.(type) {
	case starlark.String:
		return string(val)
	case starlark.Int:
		i, _ := val.Int64()
		return i
	case starlark.Float:
		return float64(val)
	case starlark.Bool:
		return bool(val)
	case *starlark.List:
		result := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = fromStarlark(val.Index(i))
		}
		return result
	case *starlark.Dict:
		result := make(map[string]any)
		for _, k := range val.Keys() {
			v, _, _ := val.Get(k)
			result[string(k.(starlark.String))] = fromStarlark(v)
		}
		return result
	default:
		return v.String()
	}
}

var _ = starlarkjson.Module
var _ = json.Marshal
