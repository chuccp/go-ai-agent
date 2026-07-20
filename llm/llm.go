package llm

import (
	"sync"

	"emperror.dev/errors"
)

type LLMOptions struct {
	options map[string]any
}

type IChatService interface {
	ChatWithStream(chatMessages *ChatMessages) (*ChatResponse, error)
}

type UnifiedChatService struct {
	providerMap map[string]IChatService
	rLock       sync.RWMutex
}

func (service *UnifiedChatService) getProvider(provider string) IChatService {
	service.rLock.RLock()
	defer service.rLock.RUnlock()
	if service.providerMap == nil {
		return nil
	}
	return service.providerMap[provider]
}

func (service *UnifiedChatService) ChatWithStream(provider string, chatMessages *ChatMessages) (*ChatResponse, error) {
	chatService := service.getProvider(provider)
	if chatService == nil {
		return nil, errors.New("no such provider: " + provider)
	}
	return chatService.ChatWithStream(chatMessages)
}
func (service *UnifiedChatService) Register(provider string, chatService IChatService) {
	service.rLock.Lock()
	defer service.rLock.Unlock()
	if service.providerMap == nil {
		service.providerMap = make(map[string]IChatService)
	}
	service.providerMap[provider] = chatService
}
