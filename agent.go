package agent

import (
	"sync"

	"github.com/chuccp/go-ai-agent/llm"
)

type Chat struct {
}

func (c *Chat) ReadChatMessage() *llm.ChatStreamMessage {

	return &llm.ChatStreamMessage{}
}

type innerChat struct {
	id string
}

func (c *innerChat) getNewChat() *Chat {
	return &Chat{}
}
func (c *innerChat) SendMessage(message *llm.ChatMessage) {

}

type ChatManager struct {
	chats map[string]*innerChat
	lock  sync.RWMutex
}

func (m *ChatManager) GetChat(id string) *Chat {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if c, ok := m.chats[id]; ok {
		return c.getNewChat()
	}
	inner := &innerChat{
		id: id,
	}
	m.chats[id] = inner
	return inner.getNewChat()
}
