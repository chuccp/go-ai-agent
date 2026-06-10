package chat

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/chat/claude"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-ai-agent/chat/native/gemini"
	"github.com/chuccp/go-ai-agent/chat/native/volcengine"
	"github.com/chuccp/go-ai-agent/chat/openai"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// UnifiedChatService 统一聊天服务
type UnifiedChatService struct {
	mu               sync.RWMutex
	providers        map[uint]common.ChatProvider
	config           config.IConfig
	defaultModelPath string
}

// SetDefaultPath sets the default model path for fallback resolution.
func (s *UnifiedChatService) SetDefaultPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultModelPath = path
}

// GetDefaultPath returns the default model path.
func (s *UnifiedChatService) GetDefaultPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultModelPath
}

func NewUnifiedChatService() *UnifiedChatService {
	return &UnifiedChatService{
		providers: make(map[uint]common.ChatProvider),
	}
}

// Init 实现 IService 接口
func (s *UnifiedChatService) Init(ctx *core.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = ctx.GetConfig()
	return nil
}

func (s *UnifiedChatService) RegisterProvider(id uint, provider common.ChatProvider) error {
	configPrefix := "chat." + strconv.FormatUint(uint64(id), 10)
	switch p := provider.(type) {
	case *openai.Provider:
		p.SetConfigPrefix(configPrefix)
	case *claude.Provider:
		p.SetConfigPrefix(configPrefix)
	case *gemini.GeminiProvider:
		p.SetConfigPrefix(configPrefix)
	case *volcengine.VolcengineProvider:
		p.SetConfigPrefix(configPrefix)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[id] = provider
	return nil
}

func (s *UnifiedChatService) UnregisterProvider(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.providers, id)
}

func (s *UnifiedChatService) GetProvider(id uint) (common.ChatProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider not found for id: %d", id)
	}
	return provider, nil
}

func (s *UnifiedChatService) GetChatService(path string) (common.ChatService, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid path format, expected: id.model")
	}

	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid provider id in path: %s", parts[0])
	}

	provider, err := s.GetProvider(uint(id))
	if err != nil {
		return nil, err
	}

	return provider.GetChat(parts[1])
}

func (s *UnifiedChatService) Chat(path string, text string, options *common.LLMOptions) (string, error) {
	service, err := s.GetChatService(path)
	if err != nil {
		return "", err
	}
	return service.Chat(text, options)
}

func (s *UnifiedChatService) ChatWithContext(ctx context.Context, path string, text string, options *common.LLMOptions) (string, error) {
	service, err := s.GetChatService(path)
	if err != nil {
		return "", err
	}
	return service.ChatWithContext(ctx, text, options)
}

func (s *UnifiedChatService) ChatWithHistory(path string, history []common.ChatMessage, text string, options *common.LLMOptions) (string, error) {
	service, err := s.GetChatService(path)
	if err != nil {
		return "", err
	}
	return service.ChatWithHistory(history, text, options)
}

func (s *UnifiedChatService) ChatWithHistoryWithContext(ctx context.Context, path string, history []common.ChatMessage, text string, options *common.LLMOptions) (string, error) {
	service, err := s.GetChatService(path)
	if err != nil {
		return "", err
	}
	return service.ChatWithHistoryWithContext(ctx, history, text, options)
}

func (s *UnifiedChatService) ChatStream(path string, text string, handler *common.StreamHandler, options *common.LLMOptions) error {
	service, err := s.GetChatService(path)
	if err != nil {
		return err
	}
	return service.ChatStream(text, handler, options)
}

func (s *UnifiedChatService) ChatStreamWithContext(ctx context.Context, path string, history []common.ChatMessage, text string, handler *common.StreamHandler, options *common.LLMOptions) error {
	service, err := s.GetChatService(path)
	if err != nil {
		return err
	}
	return service.ChatStreamWithContext(ctx, history, text, handler, options)
}

func (s *UnifiedChatService) ChatWithTools(ctx context.Context, path string, history []common.ChatMessage, text string, opts *common.LLMOptions) (*common.ChatResponse, error) {
	service, err := s.GetChatService(path)
	if err != nil {
		return nil, err
	}
	return service.ChatWithTools(ctx, history, text, opts)
}

func (s *UnifiedChatService) ListAllModels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	models := make([]string, 0)
	for id, provider := range s.providers {
		for _, model := range provider.GetModels() {
			models = append(models, strconv.FormatUint(uint64(id), 10)+"."+model)
		}
	}
	return models
}

// ConfigureProvider sets API credentials and re-initializes a provider from DB-stored config.
func (s *UnifiedChatService) ConfigureProvider(id uint, name, apiKey, modelName, baseURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, ok := s.providers[id]
	if !ok {
		return fmt.Errorf("provider not found for id: %d", id)
	}

	if s.config == nil {
		return fmt.Errorf("chat service not initialized")
	}

	configKey := "chat." + strconv.FormatUint(uint64(id), 10)
	s.config.Put(configKey+".apiKey", apiKey)
	if modelName != "" {
		s.config.Put(configKey+".model", modelName)
	}
	if baseURL != "" {
		s.config.Put(configKey+".baseUrl", baseURL)
	}

	if err := provider.Init(context.Background(), s.config); err != nil {
		log.Warn("provider configure failed", zap.Uint("id", id), zap.Error(err))
		return err
	}
	log.Info("provider configured from DB", zap.Uint("id", id), zap.String("model", modelName))
	return nil
}

// NewProvider creates a provider instance for the given vendor name.
func NewProvider(name string) (common.ChatProvider, error) {
	name = strings.ToLower(name)
	if _, ok := openai.ProviderDefaults[name]; ok || name == "openai_compat" {
		return openai.NewService(name), nil
	}
	if _, ok := claude.ProviderDefaults[name]; ok || name == "claude_compat" {
		return claude.NewService(name), nil
	}
	if name == "gemini" {
		return gemini.NewGeminiService(), nil
	}
	if name == "volcengine" {
		return volcengine.NewVolcengineService(), nil
	}
	return nil, fmt.Errorf("unknown provider type: %s", name)
}

// GetAllProviderInfo returns the default model and base URL for every registered provider.
func (s *UnifiedChatService) GetAllProviderInfo() map[uint]common.ProviderInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[uint]common.ProviderInfo, len(s.providers))
	for id, p := range s.providers {
		result[id] = p.GetProviderInfo()
	}
	return result
}

// GetGroupedProviderInfo returns provider defaults grouped by API type (openai / claude / native).
// This mirrors the setup wizard's two-level selection UI.
func GetGroupedProviderInfo() map[string]map[string]common.ProviderInfo {
	result := map[string]map[string]common.ProviderInfo{
		"openai": make(map[string]common.ProviderInfo),
		"claude": make(map[string]common.ProviderInfo),
		"native": make(map[string]common.ProviderInfo),
	}
	for name, def := range openai.ProviderDefaults {
		result["openai"][name] = common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	result["openai"]["openai_compat"] = common.ProviderInfo{}
	for name, def := range claude.ProviderDefaults {
		result["claude"][name] = common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	result["claude"]["claude_compat"] = common.ProviderInfo{}
	for name, def := range gemini.ProviderDefaults {
		result["native"][name] = common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	for name, def := range volcengine.ProviderDefaults {
		result["native"][name] = common.ProviderInfo{Model: def[1], BaseURL: def[0]}
	}
	return result
}

// NewDefaultChatService creates a UnifiedChatService with no pre-registered providers.
// Providers are registered at startup from DB records via ChatRunner.Init().
func NewDefaultChatService() *UnifiedChatService {
	return NewUnifiedChatService()
}
