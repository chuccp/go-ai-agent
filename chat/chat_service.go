package chat

import (
	"context"
	"fmt"
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
	mu        sync.RWMutex
	providers map[string]common.ChatProvider
	config    config.IConfig
}

func NewUnifiedChatService() *UnifiedChatService {
	return &UnifiedChatService{
		providers: make(map[string]common.ChatProvider),
	}
}

func NewUnifiedChatServiceWithProviders(providers ...common.ChatProvider) *UnifiedChatService {
	s := NewUnifiedChatService()
	for _, p := range providers {
		_ = s.RegisterProvider(p)
	}
	return s
}

// Init 实现 IService 接口
func (s *UnifiedChatService) Init(ctx *core.Context) error {
	cfg := ctx.GetConfig()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg

	for name, provider := range s.providers {
		if err := provider.Init(ctx, cfg); err != nil {
			log.Warn("provider init deferred — kept for metadata", zap.String("provider", name), zap.Error(err))
		}
	}
	return nil
}

func (s *UnifiedChatService) RegisterProvider(provider common.ChatProvider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := provider.Name()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if s.config != nil {
		if err := provider.Init(context.Background(), s.config); err != nil {
			return fmt.Errorf("init provider %s failed: %w", name, err)
		}
	}

	s.providers[strings.ToLower(name)] = provider
	return nil
}

func (s *UnifiedChatService) GetProvider(name string) (common.ChatProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

func (s *UnifiedChatService) GetChatService(path string) (common.ChatService, error) {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid path format, expected: provider.model")
	}

	providerName := parts[0]
	model := parts[1]

	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GetChat(model)
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

func (s *UnifiedChatService) ListProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

func (s *UnifiedChatService) ListAllModels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	models := make([]string, 0)
	for name, provider := range s.providers {
		for _, model := range provider.GetModels() {
			models = append(models, name+"."+model)
		}
	}
	return models
}

// ConfigureProvider sets API credentials and re-initializes a provider from DB-stored config.
func (s *UnifiedChatService) ConfigureProvider(name, apiKey, modelName, baseURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, ok := s.providers[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	if s.config == nil {
		return fmt.Errorf("chat service not initialized")
	}

	s.config.Put("chat."+name+".apiKey", apiKey)
	if modelName != "" {
		s.config.Put("chat."+name+".model", modelName)
	}
	if baseURL != "" {
		s.config.Put("chat."+name+".baseUrl", baseURL)
	}

	if err := provider.Init(context.Background(), s.config); err != nil {
		log.Warn("provider configure failed", zap.String("provider", name), zap.Error(err))
		return err
	}
	log.Info("provider configured from DB", zap.String("provider", name), zap.String("model", modelName))
	return nil
}

// GetAllProviderInfo returns the default model and base URL for every registered provider.
func (s *UnifiedChatService) GetAllProviderInfo() map[string]common.ProviderInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]common.ProviderInfo, len(s.providers))
	for name, p := range s.providers {
		result[name] = p.GetProviderInfo()
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

// NewDefaultChatService creates a UnifiedChatService pre-registered with all built-in providers.
func NewDefaultChatService() *UnifiedChatService {
	providers := make([]common.ChatProvider, 0, 24)

	// OpenAI Chat Completions protocol — names from ProviderDefaults
	for name := range openai.ProviderDefaults {
		providers = append(providers, openai.NewService(name))
	}
	providers = append(providers, openai.NewService("openai_compat"))

	// Anthropic Messages protocol — names from ProviderDefaults
	for name := range claude.ProviderDefaults {
		providers = append(providers, claude.NewService(name))
	}
	providers = append(providers, claude.NewService("claude_compat"))

	// Native protocols
	providers = append(providers,
		gemini.NewGeminiService(),
		volcengine.NewVolcengineService(),
	)

	return NewUnifiedChatServiceWithProviders(providers...)
}
