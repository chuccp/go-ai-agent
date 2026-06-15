package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chuccp/go-ai-agent/internal/skill"
)

// UseSkill lets the agent execute a reusable skill directly from conversation.
type UseSkill struct {
	skillSvc *skill.Service
}

func (t *UseSkill) SetSkillService(svc any) {
	if s, ok := svc.(*skill.Service); ok {
		t.skillSvc = s
	}
}

func (t *UseSkill) Definition() Definition {
	return Definition{
		Name: "use_skill",
		Description: `Execute a reusable skill directly.

Use this when the user wants to use a capability that has been packaged as a skill.

Required: skill_id.
Optional: inputs (object), model (model path like "1.default").`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"skill_id": map[string]any{
					"type":        "string",
					"description": "Unique skill identifier",
				},
				"inputs": map[string]any{
					"type":        "object",
					"description": "Input values for the skill prompt templates",
				},
				"model": map[string]any{
					"type":        "string",
					"description": "Model path override (e.g. \"1.default\")",
				},
			},
			"required": []string{"skill_id"},
		},
	}
}

func (t *UseSkill) Execute(call Call) (string, error) {
	if t.skillSvc == nil {
		return "", fmt.Errorf("skill service not initialized")
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(call.Arguments), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	skillId, _ := params["skill_id"].(string)
	if skillId == "" {
		return "", fmt.Errorf("skill_id is required")
	}
	var inputs map[string]any
	if v, ok := params["inputs"].(map[string]any); ok {
		inputs = v
	}
	model, _ := params["model"].(string)

	output, err := t.skillSvc.Execute(context.Background(), skillId, inputs, model)
	if err != nil {
		return "", err
	}
	return output, nil
}
