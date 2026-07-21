package nodes

import "github.com/chuccp/go-ai-agent/internal2/flow/engine"

type StartNode struct{}

func (n *StartNode) Type() string { return TypeStart }

func (n *StartNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	return &engine.NodeOutput{Data: map[string]any{}, Status: engine.StatusSuccess}, nil
}

type EndNode struct{}

func (n *EndNode) Type() string { return TypeEnd }

func (n *EndNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	return &engine.NodeOutput{Data: map[string]any{}, Status: engine.StatusSuccess}, nil
}
