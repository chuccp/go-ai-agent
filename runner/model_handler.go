package runner

import (
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/ai/chat"
	aiTypes "github.com/chuccp/go-ai-agent/ai/types"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

func (r *ChatRunner) handleModelAction(action string, params map[string]any) (string, error) {
	switch action {
	case "list":
		return r.modelList()
	case "get":
		return r.modelGet(params)
	case "create":
		return r.modelCreate(params)
	case "update":
		return r.modelUpdate(params)
	case "delete":
		return r.modelDelete(params)
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

func (r *ChatRunner) modelList() (string, error) {
	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}
	list, err := aiModel.List()
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	if len(list) == 0 {
		return "No models configured yet.", nil
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total %d models:\n\n", len(list)))
	b.WriteString("| ID | Name | Provider | Model | Category | Default |\n")
	b.WriteString("|----|------|----------|-------|----------|--------|\n")
	for _, m := range list {
		def := ""
		if m.IsDefault {
			def = "✅"
		}
		b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
			m.Id, m.Name, m.Provider, m.Model, m.Category, def))
	}
	return b.String(), nil
}

func (r *ChatRunner) modelGet(params map[string]any) (string, error) {
	id, ok := getParamUint(params, "id")
	if !ok {
		return "", fmt.Errorf("model ID is required")
	}
	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}
	m, err := aiModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}
	return formatModelDetail(m), nil
}

func (r *ChatRunner) modelCreate(params map[string]any) (string, error) {
	name, _ := params["name"].(string)
	provider, _ := params["provider"].(string)
	model, _ := params["model"].(string)
	apiKey, _ := params["api_key"].(string)
	baseURL, _ := params["base_url"].(string)
	category, _ := params["category"].(string)

	if provider == "" || model == "" {
		return "", fmt.Errorf("provider and model cannot be empty")
	}
	if apiKey == "" {
		return "", fmt.Errorf("api_key cannot be empty")
	}
	if category == "" {
		category = aiTypes.CategoryLLM
	}
	if name == "" {
		name = provider + " " + model
	}

	inputTypes, _ := params["input_types"].(string)
	outputTypes, _ := params["output_types"].(string)
	supportsMultimodal, _ := params["supports_multimodal"].(bool)

	m := &entity.AIModel{
		Name:               name,
		Provider:           provider,
		Model:              model,
		Category:           category,
		APIKey:             apiKey,
		BaseURL:            baseURL,
		InputTypes:         inputTypes,
		OutputTypes:        outputTypes,
		SupportsMultimodal: supportsMultimodal,
	}

	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}
	if err := aiModel.Create(m); err != nil {
		return "", fmt.Errorf("create failed: %w", err)
	}

	// Activate the provider immediately
	prov, err := chat.NewProvider(m.Provider)
	if err != nil {
		log.Warn("unknown provider type", zap.String("provider", m.Provider), zap.Error(err))
	} else {
		r.chatService.RegisterProvider(m.Id, prov)
		if cfgErr := r.chatService.ConfigureProvider(m.Id, m.Provider, m.APIKey, m.Model, m.BaseURL); cfgErr != nil {
			log.Warn("provider configure failed", zap.Uint("id", m.Id), zap.Error(cfgErr))
		}
	}

	return fmt.Sprintf("Model created: #%d %s (%s.%s)", m.Id, m.Name, m.Provider, m.Model), nil
}

func (r *ChatRunner) modelUpdate(params map[string]any) (string, error) {
	id, ok := getParamUint(params, "id")
	if !ok {
		return "", fmt.Errorf("model ID is required")
	}

	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}
	m, err := aiModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	changed := false
	if v, ok := params["name"].(string); ok && v != "" {
		m.Name = v
		changed = true
	}
	if v, ok := params["provider"].(string); ok && v != "" {
		m.Provider = v
		changed = true
	}
	if v, ok := params["model"].(string); ok && v != "" {
		m.Model = v
		changed = true
	}
	if v, ok := params["api_key"].(string); ok && v != "" {
		m.APIKey = v
		changed = true
	}
	if v, ok := params["base_url"].(string); ok && v != "" {
		m.BaseURL = v
		changed = true
	}
	if v, ok := params["category"].(string); ok && v != "" {
		m.Category = v
		changed = true
	}
	if v, ok := params["supports_multimodal"].(bool); ok {
		m.SupportsMultimodal = v
		changed = true
	}
	if !changed {
		return "No fields to update.", nil
	}

	if err := aiModel.Update(m); err != nil {
		return "", fmt.Errorf("update failed: %w", err)
	}

	// Reconfigure provider
	r.chatService.ConfigureProvider(m.Id, m.Provider, m.APIKey, m.Model, m.BaseURL)

	return fmt.Sprintf("Model #%d updated.", m.Id), nil
}

func (r *ChatRunner) modelDelete(params map[string]any) (string, error) {
	id, ok := getParamUint(params, "id")
	if !ok {
		return "", fmt.Errorf("model ID is required")
	}

	aiModel := r.aiModel()
	if aiModel == nil {
		return "", fmt.Errorf("AI model not initialized")
	}

	// Look up before deleting to confirm existence
	m, err := aiModel.FindById(id)
	if err != nil {
		return "", fmt.Errorf("query failed: %w", err)
	}

	if err := aiModel.Delete(id); err != nil {
		return "", fmt.Errorf("delete failed: %w", err)
	}
	r.chatService.UnregisterProvider(id)

	return fmt.Sprintf("Model #%d (%s) deleted.", id, m.Name), nil
}

func (r *ChatRunner) aiModel() *model.AIModelModel {
	return core.GetModel[*model.AIModelModel](r.ctx)
}

func getParamUint(params map[string]any, key string) (uint, bool) {
	v, ok := params[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return uint(n), true
	case int:
		return uint(n), true
	case int64:
		return uint(n), true
	}
	return 0, false
}

func formatModelDetail(m *entity.AIModel) string {
	def := ""
	if m.IsDefault {
		def = "Yes"
	} else {
		def = "No"
	}
	return fmt.Sprintf(`Model Details:
  ID:       %d
  Name:     %s
  Provider: %s
  Model:    %s
  Category: %s
  API Key:  ****
  Base URL: %s
  Default:  %s`,
		m.Id, m.Name, m.Provider, m.Model, m.Category, m.BaseURL, def)
}
