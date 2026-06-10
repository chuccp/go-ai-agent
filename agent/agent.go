package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/agent/tool"
	"github.com/chuccp/go-ai-agent/chat"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

const MaxIterations = 10

// ==================== Message (对话消息) ====================

// Message 对话消息，支持工具调用和工具结果
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	// Assistant 消息的工具调用
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// 工具执行结果（附在 assistant 消息上）
	ToolResults []ToolResult `json:"tool_results,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

// ToolResult 工具执行结果
type ToolResult struct {
	ToolID   string `json:"tool_id"`
	Name     string `json:"name"`
	Result   string `json:"result"`
	Success  bool   `json:"success"`
	Duration int64  `json:"duration_ms"`
}

// ==================== Event (事件) ====================

// Event Agent 事件
type Event struct {
	Type           string `json:"type"`
	Content        string `json:"content,omitempty"`
	Reasoning      string `json:"reasoning,omitempty"`
	Message        string `json:"message,omitempty"`
	Done           bool   `json:"done,omitempty"`
	Iteration      int    `json:"iteration"`
	ConversationID string `json:"conversation_id"`
	Timestamp      int64  `json:"timestamp"`
	ToolName       string `json:"tool_name,omitempty"`
	ToolInput      string `json:"tool_input,omitempty"`
	ToolOutput     string `json:"tool_output,omitempty"`
}

// Sender 消息发送接口
type Sender interface {
	Send(event Event)
}

// ==================== Chat (对话管理) ====================

// Chat 单次对话的 agent 状态
type Chat struct {
	ctx            context.Context
	id             string   // 对话 ID
	path           string   // provider.model
	opts           *common.LLMOptions
	svc            *chat.UnifiedChatService
	conn           Sender

	messages []Message   // 完整消息历史
	mu       sync.Mutex

	iteration int
}

// NewChat 创建 agent 对话
func NewChat(ctx context.Context, id string, path string, service *chat.UnifiedChatService, opts *common.LLMOptions, conn Sender) *Chat {
	return &Chat{
		ctx:      ctx,
		id:       id,
		path:     path,
		svc:      service,
		opts:     opts,
		conn:     conn,
		messages: make([]Message, 0),
	}
}

// AddUserMessage 添加用户消息
func (c *Chat) AddUserMessage(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	})
}

// SetIteration 设置起始 iteration（基于会话已有轮次）
func (c *Chat) SetIteration(n int) {
	c.iteration = n
}

// LoadHistory 加载历史消息
func (c *Chat) LoadHistory(history []common.ChatMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range history {
		c.messages = append(c.messages, Message{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: time.Now(),
		})
	}
}

// Process 处理消息（agent 主循环）
func (c *Chat) Process() {
	if c.iteration == 0 {
		// 首次 Process 才重置为 0，保持 SetIteration 的覆盖
	}
	c.opts.SetTools(tool.List())

	for c.iteration < MaxIterations {
		if c.ctx.Err() != nil {
			return
		}

		log.Info("Agent iteration",
			zap.Int("iteration", c.iteration),
			zap.Int("msgCount", len(c.messages)))

		// 构建 LLM 消息
		llmMsgs := c.buildLLMMessages()

		resp, err := c.svc.ChatWithTools(c.ctx, c.path, llmMsgs, "", c.opts)
		if err != nil {
			c.emit(Event{Type: "error", Message: err.Error(), Done: true, Iteration: c.iteration})
			return
		}

		// 有工具调用 → 执行
		if len(resp.ToolCalls) > 0 {
			c.addToolCalls(resp)
			c.iteration++
			continue
		}

		// 无工具调用 → 输出文本，结束
		c.saveAssistantMessage(resp.Text, nil)
		if resp.Text != "" {
			c.streamText(resp.Text)
		}
		c.emit(Event{Type: "chunk", Done: true, Iteration: c.iteration})
		return
	}

	c.emit(Event{Type: "error", Message: fmt.Sprintf("max iterations (%d) reached", MaxIterations), Done: true, Iteration: c.iteration})
}

// addToolCalls 添加 assistant 消息并执行工具调用
func (c *Chat) addToolCalls(resp *common.ChatResponse) {
	c.mu.Lock()

	tcs := make([]ToolCall, len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		tcs[i] = ToolCall{ID: tc.ID, Name: tc.Name, Args: tc.Arguments}
	}

	// 保存 assistant 消息（带工具调用）
	assistantMsg := Message{
		Role:      "assistant",
		Content:   resp.Text,
		ToolCalls: tcs,
		Timestamp: time.Now(),
	}
	c.mu.Unlock()

	var results []ToolResult
	for _, tc := range resp.ToolCalls {
		log.Info("Agent tool call",
			zap.Int("iteration", c.iteration),
			zap.String("tool", tc.Name),
			zap.String("args", tc.Arguments))

		c.emit(Event{
			Type:      "tool_call",
			ToolName:  tc.Name,
			ToolInput: tc.Arguments,
			Message:   fmt.Sprintf("executing %s", tc.Name),
			Iteration: c.iteration,
		})

		start := time.Now()
		call := tool.Call{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments}
		result := tool.Execute(call)
		duration := time.Since(start).Milliseconds()

		tr := ToolResult{
			ToolID:   tc.ID,
			Name:     tc.Name,
			Result:   result.Output,
			Success:  true,
			Duration: duration,
		}
		results = append(results, tr)

		c.emit(Event{
			Type:       "tool_result",
			ToolName:   tc.Name,
			ToolOutput: result.Output,
			Message:    result.Output,
			Iteration:  c.iteration,
		})
	}

	// 将工具结果附加到 assistant 消息
	c.mu.Lock()
	assistantMsg.ToolResults = results
	c.messages = append(c.messages, assistantMsg)
	c.mu.Unlock()
}

// saveAssistantMessage 保存纯文本 assistant 消息
func (c *Chat) saveAssistantMessage(text string, tcs []ToolCall) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, Message{
		Role:      "assistant",
		Content:   text,
		ToolCalls: tcs,
		Timestamp: time.Now(),
	})
}

// buildLLMMessages 从内部消息构建发给 LLM 的消息列表
func (c *Chat) buildLLMMessages() []common.ChatMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	var result []common.ChatMessage
	for _, m := range c.messages {
		// 先处理工具结果（附加在 assistant 消息上）
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				result = append(result, common.ChatMessage{
					Role:    "user",
					Content: fmt.Sprintf("[tool_result id=%s name=%s]\n%s", tr.ToolID, tr.Name, tr.Result),
				})
			}
		}

		msg := common.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}

		// 工具调用 → 追加到 content
		if len(m.ToolCalls) > 0 {
			var calls []string
			for _, tc := range m.ToolCalls {
				calls = append(calls, fmt.Sprintf("[tool_call id=%s name=%s args=%s]", tc.ID, tc.Name, tc.Args))
			}
			msg.Content = msg.Content + "\n" + strings.Join(calls, "\n")
		}

		result = append(result, msg)
	}
	return result
}

// ==================== emit helpers ====================

func (c *Chat) emitBase() Event {
	return Event{
		Iteration:      c.iteration,
		ConversationID: c.id,
		Timestamp:      time.Now().UnixMilli(),
	}
}

func (c *Chat) emit(event Event) {
	base := c.emitBase()
	event.Iteration = base.Iteration
	event.ConversationID = base.ConversationID
	event.Timestamp = base.Timestamp
	c.conn.Send(event)
}

// streamText 将文本按字符切片流式发送，模拟真实流式输出
func (c *Chat) streamText(text string) {
	runes := []rune(text)
	chunkSize := 8
	for i := 0; i < len(runes); i += chunkSize {
		if c.ctx.Err() != nil {
			return
		}
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		c.emit(Event{Type: "chunk", Content: string(runes[i:end]), Iteration: c.iteration})
		time.Sleep(12 * time.Millisecond)
	}
}
