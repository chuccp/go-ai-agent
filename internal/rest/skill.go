package rest

import (
	"encoding/json"
	"fmt"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/skill"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type SkillRest struct {
	context     *core.Context
	skillModel  *model.SkillModel
	promptModel *model.SkillPromptModel
	resModel    *model.SkillResourceModel
	skillSvc    *skill.Service
}

func NewSkillRest() *SkillRest { return &SkillRest{} }

type SkillDetail struct {
	*entity.Skill
	Prompts   []*entity.SkillPrompt   `json:"prompts"`
	Resources []*entity.SkillResource `json:"resources"`
}

func (r *SkillRest) Init(ctx *core.Context) error {
	r.context = ctx
	r.skillModel = core.GetModel[*model.SkillModel](ctx)
	r.promptModel = core.GetModel[*model.SkillPromptModel](ctx)
	r.resModel = core.GetModel[*model.SkillResourceModel](ctx)
	r.skillSvc = core.GetService[*skill.Service](ctx)

	ctx.Get("/api/skills", r.listSkills)
	ctx.Post("/api/skills", r.createSkill)
	ctx.Get("/api/skills/:id", r.getSkill)
	ctx.Put("/api/skills/:id", r.updateSkill)
	ctx.Delete("/api/skills/:id", r.deleteSkill)
	ctx.Post("/api/skills/:id/execute", r.executeSkill)
	return nil
}

func (r *SkillRest) listSkills(req *web.Request) (any, error) {
	skills, err := r.skillModel.WithContext(req.Ctx()).Query().Order("updated_at desc").All()
	if err != nil {
		return nil, err
	}
	return web.Data(skills), nil
}

func (r *SkillRest) createSkill(req *web.Request) (any, error) {
	j, _ := req.Json()
	m := map[string]any(*j)
	s := &entity.Skill{
		SkillId:      j.GetString("skill_id"),
		Name:         j.GetString("name"),
		Version:      j.GetString("version"),
		Description:  j.GetString("description"),
		Icon:         j.GetString("icon"),
		Inputs:       j.GetString("inputs"),
		Outputs:      j.GetString("outputs"),
		DefaultModel: j.GetString("default_model"),
		Enabled:      true,
	}
	if v, ok := m["enabled"].(bool); ok {
		s.Enabled = v
	}
	if s.SkillId == "" {
		return nil, fmt.Errorf("skill_id is required")
	}
	if s.Name == "" {
		s.Name = s.SkillId
	}
	if err := r.skillModel.WithContext(req.Ctx()).Save(s); err != nil {
		return nil, err
	}
	prompts := extractSkillPrompts(m, s.Id)
	for _, p := range prompts {
		_ = r.promptModel.WithContext(req.Ctx()).Save(p)
	}
	return web.Data(s), nil
}

func (r *SkillRest) getSkill(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	s, err := r.skillModel.WithContext(req.Ctx()).FindByPK(id)
	if err != nil {
		return nil, err
	}
	prompts, _ := r.promptModel.WithContext(req.Ctx()).Query().Where("skill_id = ?", id).All()
	resources, _ := r.resModel.WithContext(req.Ctx()).Query().Where("skill_id = ?", id).All()
	return web.Data(&SkillDetail{Skill: s, Prompts: prompts, Resources: resources}), nil
}

func (r *SkillRest) updateSkill(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	s, err := r.skillModel.WithContext(req.Ctx()).FindByPK(id)
	if err != nil {
		return nil, err
	}
	j, _ := req.Json()
	m := map[string]any(*j)
	if v := j.GetString("name"); v != "" {
		s.Name = v
	}
	if v := j.GetString("version"); v != "" {
		s.Version = v
	}
	if v := j.GetString("description"); v != "" {
		s.Description = v
	}
	if v := j.GetString("icon"); v != "" {
		s.Icon = v
	}
	if v := j.GetString("inputs"); v != "" {
		s.Inputs = v
	}
	if v := j.GetString("outputs"); v != "" {
		s.Outputs = v
	}
	if v := j.GetString("default_model"); v != "" {
		s.DefaultModel = v
	}
	if v, ok := m["enabled"].(bool); ok {
		s.Enabled = v
	}

	if err := r.skillModel.WithContext(req.Ctx()).Save(s); err != nil {
		return nil, err
	}

	// Replace prompts if provided.
	if prompts := extractSkillPrompts(m, id); len(prompts) > 0 {
		_ = r.promptModel.WithContext(req.Ctx()).Delete().Where("skill_id = ?", id).Delete()
		for _, p := range prompts {
			_ = r.promptModel.WithContext(req.Ctx()).Save(p)
		}
	}
	return web.Data(s), nil
}

func (r *SkillRest) deleteSkill(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	_ = r.promptModel.WithContext(req.Ctx()).Delete().Where("skill_id = ?", id).Delete()
	_ = r.resModel.WithContext(req.Ctx()).Delete().Where("skill_id = ?", id).Delete()
	if err := r.skillModel.WithContext(req.Ctx()).DeleteByPK(id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

func (r *SkillRest) executeSkill(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	s, err := r.skillModel.WithContext(req.Ctx()).FindByPK(id)
	if err != nil {
		return nil, err
	}
	j, _ := req.Json()
	m := map[string]any(*j)
	inputs := make(map[string]any)
	if raw := j.GetString("inputs"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &inputs)
	} else if v, ok := m["inputs"].(map[string]any); ok {
		inputs = v
	}
	result, err := r.skillSvc.Execute(req.Ctx(), s.SkillId, inputs, j.GetString("model"))
	if err != nil {
		return nil, err
	}
	return web.Data(map[string]any{"output": result}), nil
}

func extractSkillPrompts(m map[string]any, skillId uint) []*entity.SkillPrompt {
	var prompts []*entity.SkillPrompt
	if raw, ok := m["prompts"]; ok {
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &prompts)
		for _, p := range prompts {
			p.Id = 0
			p.SkillId = skillId
		}
	}
	return prompts
}
