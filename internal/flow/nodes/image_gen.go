package nodes

import (
	"encoding/json"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

type ImageGenNodeConfig struct {
	Model      string `json:"model"`
	Prompt     string `json:"prompt"`
	Count      int    `json:"count"`
	AspectRatio string `json:"aspect_ratio"`
}

func (c *ImageGenNodeConfig) UnmarshalJSON(data []byte) error {
	type Alias ImageGenNodeConfig
	aux := &struct{ *Alias }{Alias: (*Alias)(c)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if c.Count == 0 {
		if v, ok := raw["max_number"].(float64); ok && v > 0 {
			c.Count = int(v)
		}
	}
	if c.AspectRatio == "" {
		if v, ok := raw["scale"].(string); ok {
			c.AspectRatio = v
		}
	}
	return nil
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
	if cfg.Count == 0 {
		cfg.Count = 1
	}

	prompt := renderPrompt(cfg.Prompt, ctx)

	args := map[string]any{
		"model":       cfg.Model,
		"prompt":      prompt,
		"count":       cfg.Count,
		"aspect_ratio": cfg.AspectRatio,
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
