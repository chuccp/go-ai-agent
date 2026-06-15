package skill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-web-frame/core"
)

// Executor runs a rendered skill prompt against an LLM.
// The actual implementation is injected by the application layer to avoid import cycles.
type Executor interface {
	Execute(ctx context.Context, modelPath, prompt string) (string, error)
}

// Service executes skills (prompt-based reusable units) either as flow nodes or via tool calls.
type Service struct {
	skillModel  *model.SkillModel
	promptModel *model.SkillPromptModel
	executor    Executor
	defaultModelPath string
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Init(ctx *core.Context) error {
	s.skillModel = core.GetModel[*model.SkillModel](ctx)
	s.promptModel = core.GetModel[*model.SkillPromptModel](ctx)
	return nil
}

// SetExecutor injects the LLM executor.
func (s *Service) SetExecutor(e Executor) {
	s.executor = e
}

// SetDefaultModelPath sets the fallback model path used when a skill has no default_model.
func (s *Service) SetDefaultModelPath(path string) {
	s.defaultModelPath = path
}

// Execute runs a skill by its human-readable skill_id (e.g. "summary-v1").
// Inputs are used to render prompt templates.
func (s *Service) Execute(ctx context.Context, skillId string, inputs map[string]any, modelPath string) (string, error) {
	skill, prompts, err := s.load(skillId)
	if err != nil {
		return "", err
	}
	_ = ctx

	if modelPath == "" {
		modelPath = skill.DefaultModel
	}
	if modelPath == "" {
		modelPath = s.defaultModelPath
	}
	if modelPath == "" {
		return "", fmt.Errorf("no model configured for skill %s", skillId)
	}
	if s.executor == nil {
		return "", fmt.Errorf("skill executor not configured")
	}

	if inputs == nil {
		inputs = make(map[string]any)
	}

	// Build prompt from skill description + all prompt files.
	var parts []string
	if skill.Description != "" {
		parts = append(parts, skill.Description)
	}
	for _, p := range prompts {
		rendered, err := renderTemplate(p.Content, inputs)
		if err != nil {
			return "", fmt.Errorf("failed to render prompt %s: %w", p.Name, err)
		}
		parts = append(parts, rendered)
	}
	prompt := strings.Join(parts, "\n\n")

	return s.executor.Execute(ctx, modelPath, prompt)
}

// Get returns the skill metadata and prompts.
func (s *Service) Get(skillId string) (*entity.Skill, []*entity.SkillPrompt, error) {
	return s.load(skillId)
}

func (s *Service) load(skillId string) (*entity.Skill, []*entity.SkillPrompt, error) {
	if s.skillModel == nil {
		return nil, nil, fmt.Errorf("skill service not initialized")
	}
	skill, err := s.skillModel.FindOne("skill_id = ?", skillId)
	if err != nil {
		return nil, nil, fmt.Errorf("skill not found: %s", skillId)
	}
	prompts, err := s.promptModel.Query().Where("skill_id = ?", skill.Id).All()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load skill prompts: %w", err)
	}
	return skill, prompts, nil
}

func renderTemplate(tmpl string, data map[string]any) (string, error) {
	if tmpl == "" {
		return "", nil
	}
	// Allow missing keys to render as empty string instead of failing.
	t, err := template.New("skill").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DecodeInputs parses a JSON schema default or a JSON object into an inputs map.
func DecodeInputs(schema string, values map[string]any) (map[string]any, error) {
	if schema == "" {
		return values, nil
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(schema), &s); err != nil {
		return nil, err
	}
	defaults := make(map[string]any)
	if props, ok := s["properties"].(map[string]any); ok {
		for k, v := range props {
			if prop, ok := v.(map[string]any); ok {
				if def, ok := prop["default"]; ok {
					defaults[k] = def
				}
			}
		}
	}
	for k, v := range values {
		defaults[k] = v
	}
	return defaults, nil
}
