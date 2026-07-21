package nodes

import (
	"fmt"

	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

// SwitchNodeConfig is the configuration for a switch node.
// The script assigns a string to 'result'. The node routes to the outgoing edge
// whose source_handle matches that string. Falls back to "default" if empty.
type SwitchNodeConfig struct {
	Script string `json:"script"` // Starlark expression returning string
}

// SwitchNode evaluates a Starlark script and routes to the matching branch.
// Example: result="cat" follows source_handle="cat" edge.
type SwitchNode struct{}

func NewSwitchNode() *SwitchNode { return &SwitchNode{} }

func (n *SwitchNode) Type() string { return TypeSwitch }

func (n *SwitchNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[SwitchNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Script == "" {
		return nil, fmt.Errorf("switch: script is required")
	}

	label, err := evalSwitch(ctx, cfg.Script)
	if err != nil {
		return nil, fmt.Errorf("switch: %w", err)
	}
	if label == "" {
		label = "default"
	}

	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput: label,
			"branch":  label,
		},
		Status:   engine.StatusSuccess,
		NextNode: label,
	}, nil
}

// evalSwitch runs a Starlark script and expects a string result.
func evalSwitch(ctx *engine.ExecutionContext, script string) (string, error) {
	thread := newStarlarkThread("switch")
	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, "switch.py", script, starlarkPredeclared(ctx))
	if err != nil {
		return "", err
	}

	val, ok := globals["result"]
	if !ok || val == starlark.None {
		return "", fmt.Errorf("switch script must assign a string to 'result'")
	}

	s, ok := val.(starlark.String)
	if !ok {
		return "", fmt.Errorf("switch script must return a string, got %s", val.Type())
	}
	return string(s), nil
}
