package runner

import (
	"context"
	"strings"
	"sync"

	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/gorilla/websocket"
)

// handleStreamChat is the legacy streaming handler (text-based history).
func (r *ChatRunner) handleStreamChat(conn *websocket.Conn, path string, history []common.ChatMessage, text string, opts *common.LLMOptions, sessionID uint) {
	handler := common.NewStreamHandler()
	var mu sync.Mutex
	var assistantContent strings.Builder

	handler.OnContent(func(content string, reasoning bool) {
		mu.Lock()
		defer mu.Unlock()
		resp := WSResponse{Type: "chunk", Content: content, Done: false}
		if reasoning {
			resp.Reasoning = content
		} else {
			assistantContent.WriteString(content)
		}
		r.sendJSON(conn, resp)
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

// handleStreamChatFull is like handleStreamChat but takes complete messages array
// including the user message (possibly with ContentParts for multi-modal).
func (r *ChatRunner) handleStreamChatFull(conn *websocket.Conn, path string, messages []common.ChatMessage, opts *common.LLMOptions, sessionID uint) {
	handler := common.NewStreamHandler()
	var mu sync.Mutex
	var assistantContent strings.Builder

	handler.OnContent(func(content string, reasoning bool) {
		mu.Lock()
		defer mu.Unlock()
		resp := WSResponse{Type: "chunk", Content: content, Done: false}
		if reasoning {
			resp.Reasoning = content
		} else {
			assistantContent.WriteString(content)
		}
		r.sendJSON(conn, resp)
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

	if err := r.chatService.ChatStreamWithContext(context.Background(), path, messages, "", handler, opts); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
	}
}

// handleNonStreamChatFull is like handleNonStreamChat but takes complete messages array.
func (r *ChatRunner) handleNonStreamChatFull(conn *websocket.Conn, path string, messages []common.ChatMessage, opts *common.LLMOptions, sessionID uint) {
	content, err := r.chatService.ChatWithHistoryWithContext(context.Background(), path, messages, "", opts)
	if err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error(), Done: true})
		return
	}
	r.sendJSON(conn, WSResponse{Type: "chunk", Content: content, Done: true})
	if sessionID > 0 && content != "" {
		r.messageModel.Create(&entity.ChatMessage{SessionId: sessionID, Role: "assistant", Content: content})
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
