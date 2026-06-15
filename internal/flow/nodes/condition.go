package nodes

import (
	"fmt"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// ConditionNodeConfig is the configuration for a condition node.
// It supports either a raw Starlark script OR a simple field/operator/value rule.
// The script has access to ctx["node_label"]["field"] for all upstream node outputs.
// It must assign a boolean to the 'result' variable: True => "true" branch, False => "false" branch.
type ConditionNodeConfig struct {
	Script   string `json:"script"`    // Starlark expression returning bool
	Field    string `json:"field"`     // e.g. "node_name.output"
	Operator string `json:"operator"`  // contains/equals/not_empty
	Value    string `json:"value"`     // compare value
}

// ConditionNode evaluates a rule or Starlark script to choose the next path.
// result=True => source_handle="true", result=False => "false"
type ConditionNode struct{}

func NewConditionNode() *ConditionNode { return &ConditionNode{} }

func (n *ConditionNode) Type() string { return "condition" }

func (n *ConditionNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ConditionNodeConfig](config)
	if err != nil {
		return nil, err
	}

	var result bool
	if cfg.Script != "" {
		result, err = evalCondition(ctx, cfg.Script)
		if err != nil {
			return nil, fmt.Errorf("condition: %w", err)
		}
	} else if cfg.Field != "" {
		raw, ok := ctx.Get(cfg.Field)
		if !ok {
			for label, output := range ctx.AllNodeOutputs() {
				if label+"."+KeyOutput == cfg.Field {
					raw = output.Data[KeyOutput]
					ok = true
					break
				}
			}
		}
		result = Evaluate(cfg.Operator, fmt.Sprintf("%v", raw), cfg.Value)
	} else {
		return nil, fmt.Errorf("condition: script or field/operator/value is required")
	}

	nextNode := "false"
	if result {
		nextNode = "true"
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: fmt.Sprintf("%v", result),
			"result":  result,
		},
		Status:   engine.StatusSuccess,
		NextNode: nextNode,
	}, nil
}

// evalCondition runs a Starlark script and expects a boolean result.
func evalCondition(ctx *engine.ExecutionContext, script string) (bool, error) {
	thread := &starlark.Thread{Name: "condition"}
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "condition.py", script, starlarkPredeclared(ctx))
	if err != nil {
		return false, err
	}

	val, ok := globals["result"]
	if !ok || val == starlark.None {
		return false, fmt.Errorf("condition script must assign a boolean to 'result'")
	}

	b, ok := val.(starlark.Bool)
	if !ok {
		return false, fmt.Errorf("condition script must return a boolean, got %s", val.Type())
	}
	return bool(b), nil
}
