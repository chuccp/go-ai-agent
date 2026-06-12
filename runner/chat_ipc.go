//go:build wails

package runner

import (
	"context"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/agent"
	"github.com/chuccp/go-ai-agent/ai/chat/common"
	"github.com/chuccp/go-ai-agent/entity"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// ── IPC Sender (agent.Sender via Wails runtime events) ──

// ipcSender implements agent.Sender by emitting Wails runtime events.
// This is used in desktop mode where WebSocket is replaced by Wails IPC.
type ipcSender struct {
	ctx       context.Context
	sessionID uint
	onChunk   func(content string, reasoning bool)
	onDone    func()
}

func newIpcSender(ctx context.Context, sessionID uint) *ipcSender {
	return &ipcSender{ctx: ctx, sessionID: sessionID}
}

func (s *ipcSender) Send(event agent.Event) {
	eventName := fmt.Sprintf("chat:%d:%s", s.sessionID, event.Type)
	wailsRuntime.EventsEmit(s.ctx, eventName, event)

	switch event.Type {
	case "chunk":
		if !event.Done && s.onChunk != nil {
			s.onChunk(event.Content, event.Reasoning != "")
		}
		if event.Done && s.onDone != nil {
			s.onDone()
		}
	case "error":
		log.Error("[IPC] agent error", zap.String("msg", event.Message))
	}
}

// ── IPC Agent Start ──

// StartAgentIPC runs the agent loop and streams results via Wails runtime events.
// sessionID=0 will auto-create a new session. Returns the session ID.
func (r *ChatRunner) StartAgentIPC(ctx context.Context, sessionID uint, modelPath string, userMessage string, thinkLevel string, flowID uint) (uint, error) {
	var history []common.ChatMessage
	if sessionID > 0 {
		msgs, err := r.messageModel.FindBySessionId(sessionID)
		if err == nil && len(msgs) > 0 {
			history = make([]common.ChatMessage, 0, len(msgs))
			for _, m := range msgs {
				history = append(history, common.ChatMessage{Role: m.Role, Content: m.Content})
			}
		}
	} else {
		session := &entity.ChatSession{Title: truncateText(userMessage, 30)}
		if err := r.sessionModel.Create(session); err == nil {
			sessionID = session.Id
		}
	}

	if sessionID > 0 {
		r.messageModel.Create(&entity.ChatMessage{
			SessionId: sessionID,
			Role:      "user",
			Content:   userMessage,
		})
	}

	if modelPath == "" {
		modelPath = r.defaultModelPath
	}

	if !r.providersLoaded {
		r.loadProvidersFromDB()
	}
	if modelPath == "" {
		wailsRuntime.EventsEmit(ctx, fmt.Sprintf("chat:%d:error", sessionID), agent.Event{
			Type:    "error",
			Message: "No model configured. Please add a model in Settings.",
			Done:    true,
		})
		return sessionID, nil
	}

	opts := common.NewLLMOptions()
	if thinkLevel != "" {
		opts.SetThinkingLevel(thinkLevel)
	}

	sender := newIpcSender(ctx, sessionID)
	chatID := fmt.Sprintf("%d", sessionID)
	c := agent.NewChat(context.Background(), chatID, modelPath, r.chatService, opts, sender)
	c.SetSystemPrompt(agentSystemPrompt)

	startIter := len(history) / 2
	c.SetIteration(startIter)
	for _, m := range history {
		c.AddUserMessage(m.Content)
	}
	c.AddUserMessage(userMessage)

	var assistantContent strings.Builder
	sender.onChunk = func(content string, reasoning bool) {
		if !reasoning {
			assistantContent.WriteString(content)
		}
	}
	sender.onDone = func() {
		if sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{
				SessionId: sessionID,
				Role:      "assistant",
				Content:   assistantContent.String(),
			})
		}
	}

	go c.Process()

	return sessionID, nil
}

func truncateText(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
