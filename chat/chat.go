package chat

import (
	"sync"

	"emperror.dev/errors"
)

type LLMOptions struct {
	options map[string]any
}

type IChatService interface {
	ChatWithStream(chatMessages *Messages) (*Response, error)
}

type UnifiedChatService struct {
	providerMap  map[string]IChatService
	rLock        *sync.RWMutex
	chatServices []IChatService
}

func NewUnifiedChatService() *UnifiedChatService {
	return &UnifiedChatService{
		providerMap:  make(map[string]IChatService),
		rLock:        new(sync.RWMutex),
		chatServices: make([]IChatService, 0),
	}
}

func (service *UnifiedChatService) getProvider(provider string) IChatService {
	service.rLock.RLock()
	defer service.rLock.RUnlock()
	if service.providerMap == nil {
		return nil
	}
	return service.providerMap[provider]
}

func (service *UnifiedChatService) ChatWithStream(provider string, chatMessages *Messages) (*Response, error) {
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
	service.chatServices = append(service.chatServices, chatService)
}
