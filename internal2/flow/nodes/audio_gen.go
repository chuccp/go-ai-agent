package nodes

import (
	"github.com/chuccp/go-ai-agent/internal2/flow/engine"
)

type AudioGenNodeConfig struct {
	Model string `json:"model"`
	Text  string `json:"text"`
	Voice string `json:"voice"`
}

type AudioGenNode struct{}

func NewAudioGenNode() *AudioGenNode { return &AudioGenNode{} }

func (n *AudioGenNode) Type() string { return TypeAudioGen }

func (n *AudioGenNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[AudioGenNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}

	text := renderPrompt(cfg.Text, ctx)

	args := map[string]any{
		"model": cfg.Model,
		"text":  text,
		"voice": cfg.Voice,
	}

	result, err := ctx.InvokeFunction("audio_generation", args)
	if err != nil {
		return nil, err
	}

	output := result[KeyOutput]
	url, _ := result["url"]

	data := map[string]any{
		KeyOutput: output,
		"url":     url,
		"prompt":  text,
	}
	return &engine.NodeOutput{Data: data, Status: engine.StatusSuccess}, nil
}
