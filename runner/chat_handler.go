package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/chuccp/go-ai-agent/agent"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/log"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ==================== 共享的会话准备逻辑 ====================

type chatPrep struct {
	modelPath   string
	history     []common.ChatMessage
	userMessage string
	sessionID   uint
	opts        *common.LLMOptions
}

func (r *ChatRunner) prepareChat(conn *websocket.Conn, req WSRequest, defaultModel, providerPrefix string) chatPrep {
	cp := chatPrep{}

	cp.modelPath = req.Model
	if cp.modelPath == "" {
		cp.modelPath = defaultModel
	} else if !strings.Contains(cp.modelPath, ".") {
		cp.modelPath = providerPrefix + cp.modelPath
	}

	cp.history = req.Messages
	if len(cp.history) > 0 {
		cp.userMessage = cp.history[len(cp.history)-1].Content
		cp.history = cp.history[:len(cp.history)-1]
	}

	cp.sessionID = req.SessionID
	if cp.sessionID > 0 {
		msgs, err := r.messageModel.FindBySessionId(cp.sessionID)
		if err == nil && len(msgs) > 0 {
			cp.history = make([]common.ChatMessage, 0, len(msgs))
			for _, m := range msgs {
				cp.history = append(cp.history, common.ChatMessage{Role: m.Role, Content: m.Content})
			}
		}
	} else {
		session := &entity.ChatSession{Title: r.generateTitle(cp.userMessage)}
		if err := r.sessionModel.Create(session); err == nil {
			cp.sessionID = session.Id
			r.sendJSON(conn, WSResponse{Type: "session_created", SessionID: cp.sessionID})
		}
	}

	if cp.sessionID > 0 {
		r.messageModel.Create(&entity.ChatMessage{
			SessionId: cp.sessionID,
			Role:      "user",
			Content:   cp.userMessage,
		})
	}

	cp.opts = common.NewLLMOptions()
	if req.Options != nil {
		for k, v := range req.Options {
			cp.opts.Set(k, v)
		}
	}
	return cp
}

// ==================== chat / agent 处理 ====================

func (r *ChatRunner) handleChat(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req, "volcengine.default", "volcengine.")
	if req.Stream {
		r.handleStreamChat(conn, cp.modelPath, cp.history, cp.userMessage, cp.opts, cp.sessionID)
	} else {
		r.handleNonStreamChat(conn, cp.modelPath, cp.history, cp.userMessage, cp.opts, cp.sessionID)
	}
}

func (r *ChatRunner) handleAgent(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req, "deepseek.default", "deepseek.")

	sender := &wsSender{conn: conn, runner: r}
	chatID := fmt.Sprintf("%d", cp.sessionID)
	c := agent.NewChat(context.Background(), chatID, cp.modelPath, r.chatService, cp.opts, sender)

	startIter := len(cp.history) / 2
	c.SetIteration(startIter)

	for _, m := range cp.history {
		c.AddUserMessage(m.Content)
	}
	c.AddUserMessage(cp.userMessage)

	var assistantContent strings.Builder
	sender.onChunk = func(content string, reasoning bool) {
		if !reasoning {
			assistantContent.WriteString(content)
		}
	}
	sender.onDone = func() {
		if cp.sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{
				SessionId: cp.sessionID,
				Role:      "assistant",
				Content:   assistantContent.String(),
			})
		}
	}
	c.Process()
}

func (r *ChatRunner) handleStreamChat(conn *websocket.Conn, path string, history []common.ChatMessage, text string, opts *common.LLMOptions, sessionID uint) {
	handler := common.NewStreamHandler()
	var mu sync.Mutex
	var assistantContent strings.Builder

	handler.OnContent(func(content string, reasoning bool) {
		mu.Lock()
		defer mu.Unlock()
		if !reasoning {
			assistantContent.WriteString(content)
		}
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: false})
	})

	handler.OnComplete(func(content string, reasoningStr string) {
		mu.Lock()
		defer mu.Unlock()
		r.sendJSON(conn, WSResponse{Type: "chunk", Content: "", Done: true})
		if sessionID > 0 && assistantContent.Len() > 0 {
			r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: assistantContent.String()})
		}
	})

	handler.OnError(func(err error) {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	})

	if err := r.chatService.ChatStreamWithContext(context.Background(), path, history, text, handler, opts); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	}
}

func (r *ChatRunner) handleNonStreamChat(conn *websocket.Conn, path string, history []common.ChatMessage, text string, opts *common.LLMOptions, sessionID uint) {
	var content string
	var err error
	if len(history) > 0 {
		content, err = r.chatService.ChatWithHistoryWithContext(context.Background(), path, history, text, opts)
	} else {
		content, err = r.chatService.ChatWithContext(context.Background(), path, text, opts)
	}
	if err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
		return
	}
	r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: true})
	if sessionID > 0 && content != "" {
		r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: content})
	}
}

// ==================== wsSender ====================

type wsSender struct {
	conn    *websocket.Conn
	runner  *ChatRunner
	onChunk func(content string, reasoning bool)
	onDone  func()
}

func (s *wsSender) Send(event agent.Event) {
	resp := WSResponse{
		Type:           event.Type,
		Content:        event.Content,
		Reasoning:      event.Reasoning,
		Message:        event.Message,
		Done:           event.Done,
		Iteration:      event.Iteration,
		ConversationID: event.ConversationID,
	}
	switch event.Type {
	case "chunk":
		if !event.Done && s.onChunk != nil {
			s.onChunk(event.Content, event.Reasoning != "")
		}
		if event.Done && s.onDone != nil {
			s.onDone()
		}
	case "tool_call":
		resp.Message = fmt.Sprintf("🔧 %s(%s)", event.ToolName, event.ToolInput)
	case "tool_result":
		resp.Message = fmt.Sprintf("📋 %s", event.Message)
	}
	s.runner.sendJSON(s.conn, resp)
}

// ==================== 工具方法 ====================

func (r *ChatRunner) sendJSON(conn *websocket.Conn, resp WSResponse) {
	data, err := sonic.Marshal(resp)
	if err != nil {
		log.Error("序列化 WS 响应失败", zap.Error(err))
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Error("发送 WS 消息失败", zap.Error(err))
	}
}

func (r *ChatRunner) generateTitle(text string) string {
	if len(text) > 30 {
		return text[:30] + "..."
	}
	return text
}
