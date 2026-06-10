package rest

import (
	"github.com/chuccp/go-ai-agent/chat"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
)

type ModelRest struct {
	context *core.Context
	aiModel *model.AIModelModel
}

func NewModelRest() *ModelRest { return &ModelRest{} }

func (r *ModelRest) Init(ctx *core.Context) error {
	r.context = ctx
	r.aiModel = core.GetModel[*model.AIModelModel](ctx)

	r.context.Get("/api/ai-models/providers", r.getProviders)
	r.context.Get("/api/ai-models", r.listModels)
	r.context.Post("/api/ai-models", r.createModel)
	r.context.Put("/api/ai-models/:id", r.updateModel)
	r.context.Delete("/api/ai-models/:id", r.deleteModel)
	r.context.Put("/api/ai-models/:id/default", r.setDefault)
	r.context.Put("/api/ai-models/:id/base", r.setBase)

	log.Info("AI 模型管理 REST 已初始化")
	return nil
}

func (r *ModelRest) getProviders(_ *web.Request) (any, error) {
	return web.Data(chat.GetGroupedProviderInfo()), nil
}

func (r *ModelRest) listModels(req *web.Request) (any, error) {
	category := req.Query("category")
	if category != "" {
		models, err := r.aiModel.ListByCategory(category)
		if err != nil {
			return nil, err
		}
		return web.Data(models), nil
	}
	models, err := r.aiModel.List()
	if err != nil {
		return nil, err
	}
	return web.Data(models), nil
}

func (r *ModelRest) createModel(req *web.Request) (any, error) {
	j, _ := req.Json()
	isDefault := jsonBool(j, "is_default")
	isBase := jsonBool(j, "is_base")
	m := &entity.AIModel{
		Name:        j.GetString("name"),
		Provider:    j.GetString("provider"),
		Model:       j.GetString("model"),
		Category:    j.GetString("category"),
		APIKey:      j.GetString("api_key"),
		BaseURL:     j.GetString("base_url"),
		IsDefault:   isDefault,
		IsBase:      isBase,
		Description: j.GetString("description"),
	}
	if m.Category == "" {
		m.Category = "llm"
	}
	if m.IsDefault {
		r.aiModel.ClearDefaultByCategory(m.Category)
	}
	if m.IsBase {
		r.aiModel.ClearBase()
	}
	if err := r.aiModel.Create(m); err != nil {
		return nil, err
	}
	return web.Data(m), nil
}

func (r *ModelRest) updateModel(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.FindById(id)
	if err != nil {
		return nil, err
	}
	j, _ := req.Json()
	if v := j.GetString("name"); v != "" {
		m.Name = v
	}
	if v := j.GetString("provider"); v != "" {
		m.Provider = v
	}
	if v := j.GetString("model"); v != "" {
		m.Model = v
	}
	if v := j.GetString("category"); v != "" {
		m.Category = v
	}
	if v := j.GetString("api_key"); v != "" {
		m.APIKey = v
	}
	if v := j.GetString("base_url"); v != "" {
		m.BaseURL = v
	}
	if v := j.GetString("description"); v != "" {
		m.Description = v
	}
	if isDefault := jsonBool(j, "is_default"); isDefault {
		r.aiModel.ClearDefaultByCategory(m.Category)
		m.IsDefault = true
	}
	if _, ok := (*j)["is_base"]; ok {
		if isBase := jsonBool(j, "is_base"); isBase {
			r.aiModel.ClearBase()
		}
		m.IsBase = jsonBool(j, "is_base")
	}
	if err := r.aiModel.Update(m); err != nil {
		return nil, err
	}
	return web.Data(m), nil
}

func (r *ModelRest) deleteModel(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	if err := r.aiModel.Delete(id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

func (r *ModelRest) setDefault(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.FindById(id)
	if err != nil {
		return nil, err
	}
	r.aiModel.ClearDefaultByCategory(m.Category)
	m.IsDefault = true
	if err := r.aiModel.Update(m); err != nil {
		return nil, err
	}
	return web.Data(m), nil
}

func (r *ModelRest) setBase(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.FindById(id)
	if err != nil {
		return nil, err
	}
	r.aiModel.ClearBase()
	m.IsBase = true
	if err := r.aiModel.Update(m); err != nil {
		return nil, err
	}
	return web.Data(m), nil
}

func jsonBool(j *web.JsonObject, key string) bool {
	if j == nil {
		return false
	}
	v := (*j)[key]
	if v == nil {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}
