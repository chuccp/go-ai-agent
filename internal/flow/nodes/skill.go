package nodes

import (
	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

// SkillNodeConfig holds the configuration for a skill node.
// A skill is simply a prompt template + model selection.
type SkillNodeConfig struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"` // optional, falls back to default model
}

// SkillNode executes a prompt with the specified model.
type SkillNode struct{}

func NewSkillNode() *SkillNode { return &SkillNode{} }

func (n *SkillNode) Type() string { return TypeSkill }

func (n *SkillNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[SkillNodeConfig](config)
	if err != nil {
		return nil, err
	}

	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}

	prompt := renderPrompt(cfg.Prompt, ctx)

	// Call LLM via function registry
	args := map[string]any{
		"model":      cfg.Model,
		"prompt":     prompt,
		"max_tokens": 4096,
		"stream":     true,
	}

	result, err := ctx.InvokeFunction("llm", args)
	if err != nil {
		return nil, err
	}

	output, _ := result[KeyOutput].(string)
	return &engine.NodeOutput{
		Data:   map[string]any{KeyOutput: output, KeyPrompt: prompt},
		Status: engine.StatusSuccess,
	}, nil
}
