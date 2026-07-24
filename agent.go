package agent

import (
	"sync"

	"github.com/chuccp/go-ai-agent/chat"
	"github.com/chuccp/go-ai-agent/util"
)

const (
	EventTypeChunk = "chunk"
	EventTypeError = "error"
	EventTypeDone  = "done"
)

type Event struct {
	Type           string `json:"type"`
	Content        string `json:"content,omitempty"`
	Done           bool   `json:"done,omitempty"`
	Message        string `json:"message,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type IChat interface {
	SendMessage(message *chat.Message) error
	ReadEvent() *Event
}

type Chat struct {
	chat IChat
}

func (c *Chat) SendText(message string) error {
	msg := chat.Text(message)
	return c.chat.SendMessage(&msg)
}

func (c *Chat) ReadEvent() *Event {
	return c.chat.ReadEvent()
}

type innerChat struct {
	id       string
	mu       sync.Mutex
	revQueue *util.SliceQueue
	events   *util.SliceQueue
	isRun    bool
}

func (c *innerChat) getNewChat() *Chat {
	return &Chat{chat: c}
}

func (c *innerChat) SendMessage(message *chat.Message) error {
	err := c.revQueue.Write(message)
	if err != nil {
		return err
	}
	return nil
}

func (c *innerChat) ReadEvent() *Event {
	//v, ok := c.events.Dequeue()
	//if !ok {
	//	return nil
	//}
	//return v.(*Event)
	return nil
}

func (c *innerChat) processLoop() {
	//for {
	//	v, ok := c.queue.Dequeue()
	//	if !ok {
	//		return
	//	}
	//	c.handleMessage(v.(*chat.Message))
	//}
}

func (c *innerChat) handleMessage(msg *chat.Message) {
	//m := *msg
	//c.mu.Lock()
	//c.messages = append(c.messages, m)
	//msgs := make([]chat.Message, len(c.messages))
	//copy(msgs, c.messages)
	//c.mu.Unlock()
	//
	//if c.manager.defaultChatService == nil {
	//	c.events.Offer(&Event{Type: EventTypeError, Message: "no chat service configured", Done: true})
	//	return
	//}
	//
	//req := &chat.Messages{
	//	Model:     c.manager.defaultModel,
	//	Messages:  msgs,
	//	MaxTokens: 4096,
	//	Stream:    true,
	//}
	//
	//resp, err := c.manager.defaultChatService.ChatWithStream(req)
	//if err != nil {
	//	c.events.Offer(&Event{Type: EventTypeError, Message: err.Error(), Done: true})
	//	return
	//}
	//
	//var sb strings.Builder
	//for evt := resp.ReadEvent(); evt != nil; evt = resp.ReadEvent() {
	//	switch e := evt.(type) {
	//	case *chat.ContentBlockDeltaEvent:
	//		if e.Delta.Type == "text_delta" {
	//			sb.WriteString(e.Delta.Text)
	//			c.events.Offer(&Event{
	//				Type:           EventTypeChunk,
	//				Content:        e.Delta.Text,
	//				ConversationID: c.id,
	//			})
	//		}
	//	case *chat.MessageStopEvent:
	//		fullText := sb.String()
	//		c.mu.Lock()
	//		c.messages = append(c.messages, chat.Message{
	//			Role:    chat.RoleAssistant,
	//			Content: []chat.ContentBlock{{Type: chat.ContentTypeText, Text: fullText}},
	//		})
	//		c.mu.Unlock()
	//		c.events.Offer(&Event{
	//			Type:           EventTypeDone,
	//			Done:           true,
	//			ConversationID: c.id,
	//		})
	//	case *chat.ErrorEvent:
	//		c.events.Offer(&Event{
	//			Type:           EventTypeError,
	//			Message:        e.Error(),
	//			ConversationID: c.id,
	//		})
	//	}
	//}
}

type ChatManager struct {
	chats              map[string]*innerChat
	lock               *sync.RWMutex
	unifiedChatService *chat.UnifiedChatService
	defaultChatService chat.IChatService
	defaultModel       string
	defaultProvider    string
}

func NewChatManager() *ChatManager {
	return &ChatManager{
		chats:              make(map[string]*innerChat),
		lock:               new(sync.RWMutex),
		unifiedChatService: chat.NewUnifiedChatService(),
	}
}

func (m *ChatManager) RegisterLLM(provider string, chatService chat.IChatService, model string, isDefault bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.unifiedChatService.Register(provider, chatService)
	if isDefault {
		m.defaultChatService = chatService
		m.defaultModel = model
		m.defaultProvider = provider
	}
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
	if c, ok = m.chats[id]; ok {
		return c.getNewChat()
	}
	inner := &innerChat{
		id: id,
		//manager: m,
	}
	//inner.process()
	m.chats[id] = inner
	return inner.getNewChat()
}
