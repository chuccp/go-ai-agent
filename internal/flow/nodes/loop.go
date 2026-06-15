package nodes

import (
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// LoopNodeConfig Loop node config
type LoopNodeConfig struct {
	MaxIterations int    `json:"max_iterations"` // Max iterations, 0=unlimited (max 100)
	BreakField    string `json:"break_field"`    // Break condition check field, e.g. "child.output"
	BreakOperator string `json:"break_operator"` // contains/equals/not_empty
	BreakValue    string `json:"break_value"`    // Break compare value
}

// LoopNode Loop over child nodes until break condition is met or max iterations reached
type LoopNode struct{}

func NewLoopNode() *LoopNode { return &LoopNode{} }

func (n *LoopNode) Type() string { return "loop" }

func (n *LoopNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[LoopNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10 // Default max 10 iterations
	}
	if cfg.MaxIterations > 100 {
		cfg.MaxIterations = 100
	}

	// Requires Engine reference to run sub-flow
	// Loop node depends on engine.runSubFlow -- called by Engine.Run when children are detected
	// Returns placeholder output, actual loop handled by Engine
	return &engine.NodeOutput{
		Data: map[string]any{
			KeyOutput:       "",
			"max_iterations": cfg.MaxIterations,
		},
		Status: engine.StatusSuccess,
	}, nil
}

// LoopContext Loop execution context (used by Engine)
type LoopContext struct {
	MaxIterations int
	BreakField    string
	BreakOperator string
	BreakValue    string
	CurrentIter   int
}

// ShouldBreak Check whether loop should break
func (lc *LoopContext) ShouldBreak(ctx *engine.ExecutionContext) bool {
	lc.CurrentIter++
	if lc.CurrentIter >= lc.MaxIterations {
		return true
	}
	if lc.BreakField == "" {
		return false
	}

	field := lc.BreakField
	if !strings.Contains(field, ".") {
		field = field + "." + KeyOutput
	}
	raw, ok := ctx.Get(field)
	if !ok {
		for label, output := range ctx.AllNodeOutputs() {
			if label+"."+KeyOutput == field {
				raw = output.Data[KeyOutput]
				ok = true
				break
			}
		}
	}
	if !ok {
		return false
	}

	fieldVal := fmt.Sprintf("%v", raw)
	return Evaluate(lc.BreakOperator, fieldVal, lc.BreakValue)
}
