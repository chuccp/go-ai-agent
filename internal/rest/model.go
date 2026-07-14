package rest

import (
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	aiTypes "github.com/chuccp/go-ai-agent/internal/ai/types"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
	"go.uber.org/zap"
)

type ModelRest struct {
	context     *core.Context
	aiModel     *model.AIModelModel
	chatService *chat.UnifiedChatService
}

func NewModelRest() *ModelRest { return &ModelRest{} }

func (r *ModelRest) Init(ctx *core.Context) error {
	r.context = ctx
	r.aiModel = core.GetModel[*model.AIModelModel](ctx)
	r.chatService = core.GetService[*chat.UnifiedChatService](ctx)

	r.context.Get("/api/ai-models/providers", r.getProviders)
	r.context.Get("/api/ai-models", r.listModels)
	r.context.Post("/api/ai-models", r.createModel)
	r.context.Put("/api/ai-models/:id", r.updateModel)
	r.context.Delete("/api/ai-models/:id", r.deleteModel)
	r.context.Put("/api/ai-models/:id/default", r.setDefault)
	r.context.Put("/api/ai-models/:id/base", r.setBase)

	log.Info("AI Model management REST initialized")
	return nil
}

func (r *ModelRest) getProviders(_ *web.Request) (any, error) {
	return web.Data(chat.GetGroupedProviderInfo()), nil
}

func (r *ModelRest) listModels(req *web.Request) (any, error) {
	category := req.Query("category")
	if category != "" {
		models, err := r.aiModel.WithContext(req.Ctx()).ListByCategory(category)
		if err != nil {
			return nil, err
		}
		maskModels(models)
		return web.Data(models), nil
	}
	models, err := r.aiModel.WithContext(req.Ctx()).List()
	if err != nil {
		return nil, err
	}
	maskModels(models)
	return web.Data(models), nil
}

// maskModels masks the API key on every model in the slice.
func maskModels(models []*entity.AIModel) {
	for _, m := range models {
		if m != nil {
			m.APIKey = maskAPIKey(m.APIKey)
		}
	}
}

// maskAPIKey masks an API key for safe display.
// Keeps the first 4 and last 4 characters, replacing the middle with "****".
// If the key is shorter than 12 characters it is fully masked as "****".
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) < 12 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func (r *ModelRest) createModel(req *web.Request) (any, error) {
	j, _ := req.Json()
	m := &entity.AIModel{
		Name:               j.GetString("name"),
		Provider:           j.GetString("provider"),
		Model:              j.GetString("model"),
		Category:           j.GetString("category"),
		APIKey:             j.GetString("api_key"),
		BaseURL:            j.GetString("base_url"),
		IsDefault:          j.GetBool("is_default"),
		IsBase:             j.GetBool("is_base"),
		Description:        j.GetString("description"),
		InputTypes:         j.GetString("input_types"),
		OutputTypes:        j.GetString("output_types"),
		SupportsMultimodal: j.GetBool("supports_multimodal"),
		ThinkingLevel:      j.GetString("thinking_level"),
	}
	if m.Category == "" {
		m.Category = aiTypes.CategoryLLM
	}
	if m.IsDefault {
		r.aiModel.WithContext(req.Ctx()).ClearDefaultByCategory(m.Category)
	}
	if m.IsBase {
		r.aiModel.WithContext(req.Ctx()).ClearBase()
	}
	if err := r.aiModel.WithContext(req.Ctx()).Create(m); err != nil {
		return nil, err
	}
	// Activate provider immediately if system is initialized
	if r.context.GetConfig().GetBoolOrDefault("system.init", false) {
		r.activateModel(m)
	}
	m.APIKey = maskAPIKey(m.APIKey)
	return web.Data(m), nil
}

func (r *ModelRest) activateModel(m *entity.AIModel) {
	provider, err := chat.NewProvider(m.Provider)
	if err != nil {
		log.Warn("unknown provider type", zap.String("provider", m.Provider), zap.Error(err))
		return
	}
	r.chatService.RegisterProvider(m.Id, provider)
	if err := r.chatService.ConfigureProvider(m.Id, m.Provider, m.APIKey, m.Model, m.BaseURL); err != nil {
		log.Warn("provider configure failed", zap.Uint("id", m.Id), zap.Error(err))
	}
}

func (r *ModelRest) updateModel(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.WithContext(req.Ctx()).FindById(id)
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
	if v := j.GetString("input_types"); v != "" {
		m.InputTypes = v
	}
	if v := j.GetString("output_types"); v != "" {
		m.OutputTypes = v
	}
	if j.Has("supports_multimodal") {
		m.SupportsMultimodal = j.GetBool("supports_multimodal")
	}
	if v := j.GetString("thinking_level"); v != "" {
		m.ThinkingLevel = v
	}
	if j.GetBool("is_default") {
		r.aiModel.WithContext(req.Ctx()).ClearDefaultByCategory(m.Category)
		m.IsDefault = true
	}
	if j.Has("is_base") {
		if j.GetBool("is_base") {
			r.aiModel.WithContext(req.Ctx()).ClearBase()
		}
		m.IsBase = j.GetBool("is_base")
	}
	if err := r.aiModel.WithContext(req.Ctx()).Update(m); err != nil {
		return nil, err
	}
	// Reconfigure provider if system is initialized
	if r.context.GetConfig().GetBoolOrDefault("system.init", false) {
		r.chatService.ConfigureProvider(m.Id, m.Provider, m.APIKey, m.Model, m.BaseURL)
	}
	m.APIKey = maskAPIKey(m.APIKey)
	return web.Data(m), nil
}

func (r *ModelRest) deleteModel(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	if err := r.aiModel.WithContext(req.Ctx()).Delete(id); err != nil {
		return nil, err
	}
	r.chatService.UnregisterProvider(id)
	return web.Ok("deleted"), nil
}

func (r *ModelRest) setDefault(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}
	r.aiModel.WithContext(req.Ctx()).ClearDefaultByCategory(m.Category)
	m.IsDefault = true
	if err := r.aiModel.WithContext(req.Ctx()).Update(m); err != nil {
		return nil, err
	}
	m.APIKey = maskAPIKey(m.APIKey)
	return web.Data(m), nil
}

func (r *ModelRest) setBase(req *web.Request) (any, error) {
	id := req.ParamUint("id")
	m, err := r.aiModel.WithContext(req.Ctx()).FindById(id)
	if err != nil {
		return nil, err
	}
	r.aiModel.WithContext(req.Ctx()).ClearBase()
	m.IsBase = true
	if err := r.aiModel.WithContext(req.Ctx()).Update(m); err != nil {
		return nil, err
	}
	m.APIKey = maskAPIKey(m.APIKey)
	return web.Data(m), nil
}
