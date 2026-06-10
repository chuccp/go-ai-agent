package nodes

import (
	"github.com/chuccp/go-ai-agent/flow/engine"
)

type VideoGenNodeConfig struct {
	Model    string `json:"model"`
	Prompt   string `json:"prompt"`
	Duration int    `json:"duration"`
}

type VideoGenNode struct{}

func NewVideoGenNode() *VideoGenNode { return &VideoGenNode{} }

func (n *VideoGenNode) Type() string { return TypeVideoGen }

func (n *VideoGenNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[VideoGenNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}

	prompt := renderPrompt(cfg.Prompt, ctx)

	args := map[string]any{
		"model":    cfg.Model,
		"prompt":   prompt,
		"duration": cfg.Duration,
	}

	result, err := ctx.InvokeFunction("video_generation", args)
	if err != nil {
		return nil, err
	}

	output := result[KeyOutput]
	url, _ := result["url"]

	data := map[string]any{
		KeyOutput: output,
		"url":     url,
		"prompt":  prompt,
	}
	return &engine.NodeOutput{Data: data, Status: engine.StatusSuccess}, nil
}
