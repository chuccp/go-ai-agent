package runner

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/gorilla/websocket"
)

type chatPrep struct {
	modelPath    string
	history      []common.ChatMessage
	userMessage  string
	sessionID    uint
	opts         *common.LLMOptions
	contentParts []common.ContentPart
}

func (r *ChatRunner) prepareChat(conn *websocket.Conn, req WSRequest) chatPrep {
	cp := chatPrep{}

	cp.modelPath = req.Model
	if cp.modelPath == "" {
		cp.modelPath = r.defaultModelPath
	}

	cp.history = req.Messages
	if len(cp.history) > 0 {
		cp.userMessage = cp.history[len(cp.history)-1].Content
		cp.history = cp.history[:len(cp.history)-1]
	}

	if len(req.Attachments) > 0 {
		var extraText string
		var err error
		cp.contentParts, extraText, err = r.processAttachments(req.Attachments, cp.modelPath)
		if err != nil {
			r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
			return cp
		}
		if extraText != "" {
			cp.userMessage += "\n\n" + extraText
		}
		cp.userMessage = strings.TrimSpace(cp.userMessage)
	}

	cp.sessionID = req.SessionID
	if cp.sessionID > 0 {
		msgs, err := r.messageModel.FindBySessionId(cp.sessionID)
		if err == nil && len(msgs) > 0 {
			cp.history = make([]common.ChatMessage, 0, len(msgs))
			for _, m := range msgs {
				cm := common.ChatMessage{Role: m.Role, Content: m.Content}
				if m.ToolCalls != "" {
					var tcs []common.ToolCall
					if json.Unmarshal([]byte(m.ToolCalls), &tcs) == nil {
						cm.ToolCalls = tcs
					}
				}
				cp.history = append(cp.history, cm)
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
	if dbID := parseModelDBID(cp.modelPath); dbID > 0 {
		if m, err := r.aiModel().FindById(dbID); err == nil && m != nil {
			cp.opts.SetThinkingLevel(m.ThinkingLevel)
		}
	}
	if req.Options != nil {
		for k, v := range req.Options {
			cp.opts.Set(k, v)
		}
	}
	return cp
}

func (r *ChatRunner) generateTitle(text string) string {
	if len(text) > 30 {
		return text[:30] + "..."
	}
	return text
}

func parseModelDBID(path string) uint {
	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 0 {
		return 0
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}
