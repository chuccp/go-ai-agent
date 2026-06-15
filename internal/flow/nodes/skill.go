package nodes

import (
	"fmt"

	"github.com/chuccp/go-ai-agent/internal/flow/engine"
)

type SkillNodeConfig struct {
	SkillId      string         `json:"skill_id"`
	Model        string         `json:"model"`
	InputMapping map[string]any `json:"input_mapping"`
}

type SkillNode struct{}

func NewSkillNode() *SkillNode { return &SkillNode{} }

func (n *SkillNode) Type() string { return TypeSkill }

func (n *SkillNode) Execute(ctx *engine.ExecutionContext, config string) (*engine.NodeOutput, error) {
	cfg, err := engine.GetNodeConfig[SkillNodeConfig](config)
	if err != nil {
		return nil, err
	}
	if cfg.SkillId == "" {
		return nil, fmt.Errorf("skill node: skill_id is required")
	}

	inputs := make(map[string]any)
	// Apply explicit input mappings (values support template rendering).
	for k, v := range cfg.InputMapping {
		switch s := v.(type) {
		case string:
			inputs[k] = renderPrompt(s, ctx)
		default:
			inputs[k] = v
		}
	}
	// Fallback: expose common upstream outputs as inputs.
	if len(inputs) == 0 {
		inputs = ctx.Data
	}

	args := map[string]any{
		"skill_id": cfg.SkillId,
		"inputs":   inputs,
	}
	if cfg.Model != "" {
		args["model"] = cfg.Model
	}

	result, err := ctx.InvokeFunction("skill", args)
	if err != nil {
		return nil, err
	}

	output, _ := result[KeyOutput].(string)
	return &engine.NodeOutput{
		Data:   map[string]any{KeyOutput: output},
		Status: engine.StatusSuccess,
	}, nil
}
