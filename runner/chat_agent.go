package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/agent"
	"github.com/chuccp/go-ai-agent/ai/chat/common"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/gorilla/websocket"
)

// agentSystemPrompt is the system prompt for the agent in both WebSocket and IPC modes.
const agentSystemPrompt = `You are an AI assistant that helps users create and manage workflows (flows) and AI models.

When creating flows, every node type has REQUIRED config fields that MUST be filled in:
- llm: prompt + model (use manage_models list first)
- user_input: prompt
- condition: script (Starlark, assign bool to 'result')
- switch: script (Starlark, assign string to 'result')
- transform: template
- for_each / iterator: items_key
- loop: max_iterations
- script: script (code)
- execute: command
- split: source_key + delimiter
- image_gen / video_gen: prompt
- audio_gen: text + model

Ask the user for any required fields they haven't specified. In the DESIGN step, list the key config values for every node. Never create a node with empty required fields.`

// ── wsSender (agent → WebSocket bridge) ──

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

// ── Chat / Agent handlers ──

func (r *ChatRunner) handleChat(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req)

	userMsg := common.ChatMessage{
		Role:         "user",
		Content:      cp.userMessage,
		ContentParts: cp.contentParts,
	}
	history := append(cp.history, userMsg)

	if req.Stream {
		r.handleStreamChatFull(conn, cp.modelPath, history, cp.opts, cp.sessionID)
	} else {
		r.handleNonStreamChatFull(conn, cp.modelPath, history, cp.opts, cp.sessionID)
	}
}

func (r *ChatRunner) handleAgent(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req)

	sender := &wsSender{conn: conn, runner: r}
	chatID := fmt.Sprintf("%d", cp.sessionID)
	c := agent.NewChat(context.Background(), chatID, cp.modelPath, r.chatService, cp.opts, sender)
	c.SetSystemPrompt(agentSystemPrompt)

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

// ── JSON helpers ──

func (r *ChatRunner) sendJSON(conn *websocket.Conn, resp WSResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return
	}
}
