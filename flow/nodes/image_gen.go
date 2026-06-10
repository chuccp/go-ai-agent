package nodes

import (
	"github.com/chuccp/go-ai-agent/flow/engine"
)

type ImageGenNodeConfig struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	MaxNumber int    `json:"max_number"`
	Scale     string `json:"scale"`
}

type ImageGenNode struct{}

func NewImageGenNode() *ImageGenNode { return &ImageGenNode{} }

func (n *ImageGenNode) Type() string { return TypeImageGen }

func (n *ImageGenNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[ImageGenNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.MaxNumber == 0 {
		cfg.MaxNumber = 1
	}

	prompt := renderPrompt(cfg.Prompt, ctx)

	args := map[string]any{
		"model":      cfg.Model,
		"prompt":     prompt,
		"max_number": cfg.MaxNumber,
		"scale":      cfg.Scale,
	}

	result, err := ctx.InvokeFunction("image_generation", args)
	if err != nil {
		return nil, err
	}

	output := result[KeyOutput]
	urls, _ := result["urls"]
	count, _ := result["count"]

	data := map[string]any{
		KeyOutput: output,
		"urls":    urls,
		"count":   count,
		"prompt":  prompt,
	}
	return &engine.NodeOutput{Data: data, Status: engine.StatusSuccess}, nil
}
