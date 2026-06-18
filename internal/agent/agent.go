package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/internal/agent/tool"
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

const MaxIterations = 10

// ==================== Message ====================

// Message represents a chat message, supporting tool calls and tool results
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	// Tool calls on an assistant message
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Tool execution results (attached to assistant message)
	ToolResults []ToolResult `json:"tool_results,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// ToolCall represents a tool call invocation
type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

// ToolResult represents the result of executing a tool call
type ToolResult struct {
	ToolID   string `json:"tool_id"`
	Name     string `json:"name"`
	Result   string `json:"result"`
	Success  bool   `json:"success"`
	Duration int64  `json:"duration_ms"`
}

// ==================== Event ====================

// Event is an agent event emitted during processing
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

// Sender is the interface for sending events
type Sender interface {
	Send(event Event)
}

// ==================== Chat ====================

// Chat manages the agent state for a single conversation
type Chat struct {
	ctx            context.Context
	id             string   // Conversation ID
	path           string   // provider.model
	opts           *common.LLMOptions
	svc            *chat.UnifiedChatService
	conn           Sender
	toolRegistry    *tool.Registry

	messages []Message   // Full message history
	mu       sync.Mutex

	iteration    int
	systemPrompt string
}

// NewChat creates a new agent conversation.
// It resolves UnifiedChatService and ToolRegistry from the given core.Context.
func NewChat(appCtx *core.Context, id string, path string, opts *common.LLMOptions, conn Sender) *Chat {
	return &Chat{
		ctx:          context.Background(),
		id:           id,
		path:         path,
		svc:          core.GetService[*chat.UnifiedChatService](appCtx),
		opts:         opts,
		conn:         conn,
		toolRegistry: core.GetService[*tool.Registry](appCtx),
		messages:     make([]Message, 0),
	}
}

// AddUserMessage appends a user message to the history
func (c *Chat) AddUserMessage(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, Message{
		Role:      "user",
		Content:   content,
		Timestamp: time.Now(),
	})
}

// AddMessage appends a message with the given role, content, and optional tool calls
func (c *Chat) AddMessage(role, content string, toolCalls []common.ToolCall) {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
	if len(toolCalls) > 0 {
		tcs := make([]ToolCall, len(toolCalls))
		for i, tc := range toolCalls {
			tcs[i] = ToolCall{ID: tc.ID, Name: tc.Name, Args: tc.Arguments}
		}
		msg.ToolCalls = tcs
	}
	c.messages = append(c.messages, msg)
}

// SetIteration sets the starting iteration (based on existing conversation turns)
func (c *Chat) SetIteration(n int) {
	c.iteration = n
}

// SetSystemPrompt sets the agent's system prompt
func (c *Chat) SetSystemPrompt(prompt string) {
	c.systemPrompt = prompt
}

// LoadHistory loads historical messages preserving roles and tool calls
func (c *Chat) LoadHistory(history []common.ChatMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, m := range history {
		msg := Message{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: time.Now(),
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				tcs[i] = ToolCall{ID: tc.ID, Name: tc.Name, Args: tc.Arguments}
			}
			msg.ToolCalls = tcs
		}
		c.messages = append(c.messages, msg)
	}
}

// Process runs the agent main loop with real streaming output
func (c *Chat) Process() {
	c.opts.SetTools(c.toolRegistry.List())

	for c.iteration < MaxIterations {
		if c.ctx.Err() != nil {
			return
		}

		log.Info("Agent iteration",
			zap.Int("iteration", c.iteration),
			zap.Int("msgCount", len(c.messages)))

		llmMsgs := c.buildLLMMessages()

		streamHandler := &common.StreamHandler{
			OnContentFunc: func(content string, reasoning bool) {
				if c.ctx.Err() != nil {
					return
				}
				if reasoning {
					c.emit(Event{Type: "chunk", Content: content, Reasoning: content, Iteration: c.iteration})
				} else {
					c.emit(Event{Type: "chunk", Content: content, Iteration: c.iteration})
				}
			},
		}

		resp, err := c.svc.ChatWithToolsStream(c.ctx, c.path, llmMsgs, "", c.opts, streamHandler)
		if err != nil {
			c.emit(Event{Type: "error", Message: err.Error(), Done: true, Iteration: c.iteration})
			return
		}

		if len(resp.ToolCalls) > 0 {
			c.addToolCalls(resp)
			c.iteration++
			continue
		}

		c.saveAssistantMessage(resp.Text, nil)
		c.emit(Event{Type: "chunk", Done: true, Iteration: c.iteration})
		return
	}

	c.emit(Event{Type: "error", Message: fmt.Sprintf("max iterations (%d) reached", MaxIterations), Done: true, Iteration: c.iteration})
}

// addToolCalls adds the assistant message and executes tool calls
func (c *Chat) addToolCalls(resp *common.ChatResponse) {
	c.mu.Lock()

	tcs := make([]ToolCall, len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		tcs[i] = ToolCall{ID: tc.ID, Name: tc.Name, Args: tc.Arguments}
	}

	// Save assistant message (with tool calls)
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
		result := c.toolRegistry.Execute(call)
		duration := time.Since(start).Milliseconds()

		tr := ToolResult{
			ToolID:   tc.ID,
			Name:     tc.Name,
			Result:   result.Output,
			Success:  result.Success,
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

	// Attach tool results to the assistant message
	c.mu.Lock()
	assistantMsg.ToolResults = results
	c.messages = append(c.messages, assistantMsg)
	c.mu.Unlock()
}

// saveAssistantMessage saves a plain text assistant message
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

// buildLLMMessages builds the list of messages to send to the LLM using native function calling protocol
func (c *Chat) buildLLMMessages() []common.ChatMessage {
	c.mu.Lock()
	defer c.mu.Unlock()

	var result []common.ChatMessage

	if c.systemPrompt != "" {
		result = append(result, common.ChatMessage{
			Role:    "system",
			Content: c.systemPrompt,
		})
	}

	for _, m := range c.messages {
		// Emit tool results as tool-role messages (native protocol)
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				result = append(result, common.ChatMessage{
					Role:       "tool",
					Content:    tr.Result,
					ToolCallID: tr.ToolID,
					Name:       tr.Name,
				})
			}
		}

		msg := common.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}

		// Keep tool calls as native structure on assistant messages
		if len(m.ToolCalls) > 0 {
			tcs := make([]common.ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				tcs[i] = common.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Args}
			}
			msg.ToolCalls = tcs
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
