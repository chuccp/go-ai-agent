package agent

import (
	"sync"

	"github.com/chuccp/go-ai-agent/chat"
)

type Chat struct {
}

func (c *Chat) ReadEvent() *Event {

	return &Event{}
}

type innerChat struct {
	id string
}

func (c *innerChat) getNewChat() *Chat {
	return &Chat{}
}
func (c *innerChat) SendMessage(message *chat.Message) {

}
func (c *innerChat) process() {

}

type ChatManager struct {
	chats              map[string]*innerChat
	lock               *sync.RWMutex
	unifiedChatService *chat.UnifiedChatService
}

func (m *ChatManager) RegisterLLM(provider string, chatService chat.IChatService) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.unifiedChatService.Register(provider, chatService)
}

func (m *ChatManager) GetChat(id string) *Chat {
	m.lock.RLock()
	c, ok := m.chats[id]
	m.lock.RUnlock()
	if ok {
		return c.getNewChat()
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	// 双重检查，防止并发创建
	if c, ok = m.chats[id]; ok {
		return c.getNewChat()
	}
	inner := &innerChat{
		id: id,
	}
	inner.process()
	m.chats[id] = inner
	return inner.getNewChat()
}

// Event 是 agent 内部流转的事件（非 Claude API 类型）。
type Event struct {
	Type string `json:"type"`
}
