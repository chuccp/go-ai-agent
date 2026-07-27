package agent

import (
	"strings"
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

// ChatHandler 会话处理接口，由 chatSession 实现
type ChatHandler interface {
	SendMessage(message *chat.Message) error
	ReadEvent(start uint) *EventEntry
}

// ChatClient 面向调用方的客户端句柄
type ChatClient struct {
	handler ChatHandler
	offset  uint
}

func (c *ChatClient) SendText(message string) error {
	msg := chat.Text(message)
	return c.handler.SendMessage(&msg)
}

func (c *ChatClient) ReadEvent() *Event {
	entry := c.handler.ReadEvent(c.offset)
	if entry == nil {
		return nil
	}
	c.offset = c.offset + entry.Offset
	return entry.Event
}

// EventEntry 单个事件条目，含偏移量信息
type EventEntry struct {
	Start  uint
	Offset uint
	Event  *Event
}

// eventLog 追加式事件日志
type eventLog struct {
	mu      sync.RWMutex
	entries []*EventEntry
}

func newEventLog() *eventLog {
	return &eventLog{
		entries: make([]*EventEntry, 0, 64),
	}
}

func (l *eventLog) add(event *Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, &EventEntry{
		Start:  uint(len(l.entries)),
		Offset: 1,
		Event:  event,
	})
}

// readFrom 从 start 偏移量读取下一个事件，若无新事件返回 nil
func (l *eventLog) readFrom(start uint) *EventEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if int(start) >= len(l.entries) {
		return nil
	}
	return l.entries[start]
}

// chatSession 完整会话实体，管理消息队列、对话历史和 LLM 调用
type chatSession struct {
	id                 string
	mu                 sync.Mutex
	unifiedChatService *chat.UnifiedChatService
	revQueue           *util.SliceQueueSafe[*chat.Message]
	events             *eventLog
	history            []chat.Message // 对话历史
	isRun              bool
	provider           string
}

func newChatSession(id string, unifiedChatService *chat.UnifiedChatService) *chatSession {
	return &chatSession{
		id:                 id,
		unifiedChatService: unifiedChatService,
		revQueue:           util.NewSliceQueueSafe[*chat.Message](),
		events:             newEventLog(),
		history:            make([]chat.Message, 0),
		isRun:              false,
	}
}

func (s *chatSession) newClient() *ChatClient {
	return &ChatClient{handler: s}
}

func (s *chatSession) SendMessage(message *chat.Message) error {
	err := s.revQueue.Write(message)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if !s.isRun {
		s.isRun = true
		go s.run()
	}
	s.mu.Unlock()
	return nil
}

// build 从队列中取出所有待处理消息，追加到历史记录，构建 LLM 请求
func (s *chatSession) build() *chat.Messages {
	// 从队列中读取所有新消息
	for {
		msg, err := s.revQueue.Read()
		if err != nil {
			break
		}
		s.history = append(s.history, *msg)
	}

	if len(s.history) == 0 {
		return nil
	}

	messages := &chat.Messages{
		Messages: make([]chat.Message, len(s.history)),
		Stream:   true,
	}
	copy(messages.Messages, s.history)
	return messages
}

func (s *chatSession) run() {
	defer func() {
		s.mu.Lock()
		s.isRun = false
		s.mu.Unlock()
	}()

	for {
		messages := s.build()
		if messages == nil {
			// 队列无新消息，退出 goroutine（下次 SendMessage 会重新启动）
			return
		}

		provider := s.provider
		if provider == "" {
			provider = s.unifiedChatService.DefaultProvider()
		}

		resp, err := s.unifiedChatService.ChatWithStream(provider, messages)
		if err != nil {
			s.events.add(&Event{Type: EventTypeError, Message: err.Error(), Done: true})
			return
		}

		// 消费流式事件，转换为 Event 写入 eventLog
		var sb strings.Builder
		done := false
		for evt := resp.ReadEvent(); evt != nil; evt = resp.ReadEvent() {
			switch e := evt.(type) {
			case *chat.ContentBlockDeltaEvent:
				if e.Delta.Type == "text_delta" && e.Delta.Text != "" {
					sb.WriteString(e.Delta.Text)
					s.events.add(&Event{
						Type:           EventTypeChunk,
						Content:        e.Delta.Text,
						ConversationID: s.id,
					})
				}
			case *chat.ErrorEvent:
				s.events.add(&Event{Type: EventTypeError, Message: e.Error(), Done: true})
				return
			case *chat.MessageStopEvent:
				// 模型返回 done，将完整回复追加到历史
				s.history = append(s.history, chat.Message{
					Role:    chat.RoleAssistant,
					Content: []chat.ContentBlock{{Type: chat.ContentTypeText, Text: sb.String()}},
				})
				s.events.add(&Event{Type: EventTypeDone, Done: true, ConversationID: s.id})
				done = true
			}
		}

		// 模型未正常返回 done（流异常中断），退出
		if !done {
			return
		}
		// 模型返回 done，继续循环处理队列中的下一批消息
	}
}

func (s *chatSession) ReadEvent(start uint) *EventEntry {
	return s.events.readFrom(start)
}

// ChatManager 聊天会话管理器
type ChatManager struct {
	chats              map[string]*chatSession
	lock               *sync.RWMutex
	unifiedChatService *chat.UnifiedChatService
}

func NewChatManager() *ChatManager {
	return &ChatManager{
		chats:              make(map[string]*chatSession),
		lock:               new(sync.RWMutex),
		unifiedChatService: chat.NewUnifiedChatService(),
	}
}

func (m *ChatManager) RegisterLLM(provider string, chatService chat.IChatService, isDefault bool) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.unifiedChatService.Register(provider, chatService, isDefault)
}

func (m *ChatManager) GetChat(id string) *ChatClient {
	m.lock.RLock()
	c, ok := m.chats[id]
	m.lock.RUnlock()
	if ok {
		return c.newClient()
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	if c, ok = m.chats[id]; ok {
		return c.newClient()
	}
	session := newChatSession(id, m.unifiedChatService)
	m.chats[id] = session
	return session.newClient()
}
