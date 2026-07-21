package nodes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkjson"
)

// buildStarlarkCtx builds a Starlark dict from the execution context for use in scripts.
func buildStarlarkCtx(ctx *engine.ExecutionContext) *starlark.Dict {
	d := starlark.NewDict(len(ctx.AllNodeOutputs()))
	for label, output := range ctx.AllNodeOutputs() {
		inner := starlark.NewDict(len(output.Data))
		for k, v := range output.Data {
			inner.SetKey(starlark.String(k), toStarlark(v))
		}
		d.SetKey(starlark.String(label), inner)
	}
	return d
}

// jsonParseBuiltin is the Starlark json_parse(s) builtin.
var jsonParseBuiltin = starlark.NewBuiltin("json_parse", func(
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
})

// splitBuiltin is the Starlark split(s, sep) builtin.
var splitBuiltin = starlark.NewBuiltin("split", func(
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
})

// starlarkPredeclared returns the common predeclared dict for Starlark scripts.
func starlarkPredeclared(ctx *engine.ExecutionContext) starlark.StringDict {
	return starlark.StringDict{
		"ctx":        buildStarlarkCtx(ctx),
		"json_parse": jsonParseBuiltin,
		"split":      splitBuiltin,
	}
}

var _ = starlarkjson.Module

const maxStarlarkSteps = 100000

// newStarlarkThread creates a Starlark thread with step limit to prevent infinite loops.
func newStarlarkThread(name string) *starlark.Thread {
	t := &starlark.Thread{
		Name: name,
	}
	t.SetMaxExecutionSteps(maxStarlarkSteps)
	return t
}
