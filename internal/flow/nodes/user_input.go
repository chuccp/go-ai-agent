package nodes

import (
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

type UserInputNodeConfig struct {
	Prompt      string `json:"prompt"`
	ConfirmOnly bool   `json:"confirm_only"`
}

type UserInputNode struct{}

func (n *UserInputNode) Type() string { return TypeUserInput }

func (n *UserInputNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[UserInputNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Prompt == "" {
		cfg.Prompt = "Please confirm to continue, or enter your feedback:"
	}

	prompt := renderPrompt(cfg.Prompt, ctx)

	if ctx.Emitter != nil {
		ctx.Emitter.Emit(engine.FlowEvent{
			Type:        engine.EventWaitingUser,
			ExecutionId: ctx.ExecutionId,
			Message:     prompt,
		})
	}

	response, _ := ctx.WaitUserInput()

	return &engine.NodeOutput{
		Data:    map[string]any{KeyOutput: response, KeyPrompt: prompt},
		Status:  engine.StatusSuccess,
		Message: prompt,
	}, nil
}
