package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chuccp/go-ai-agent/internal/agent"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/gorilla/websocket"
)

// agentSystemPrompt is the system prompt for the agent in both WebSocket and IPC modes.
const agentSystemPrompt = `You are go-ai-agent, an AI assistant that helps users chat, manage AI models, create and run workflows (flows), search the web, read documents, and execute local commands.

## Available Tools

| Tool | Purpose |
|------|---------|
| ask_user | Ask the user questions and block until they answer (clarify, confirm, offer choices) |
| create_flow_conversation | Start the conversational assistant to create a flow by chatting |
| manage_flows | List, search, get, delete flows |
| manage_models | List, get, create, update, delete AI model credentials |
| run_flow | Search flows by name, run/respond/status/stop flow executions |
| web_search | Search the internet for real-time information |
| read_document | Extract text from uploaded files (TXT, DOCX, XLSX, PDF) |
| execute_command | Run local shell commands (open apps, manage files, etc.) |

## Asking the User

Whenever you need clarification, confirmation, a choice, or any user input, call
the **ask_user** tool. It BLOCKS until the user responds — the frontend shows the
question UI automatically, so do NOT relay the question yourself or end your turn.
This is the canonical way to get user input; prefer it over guessing.

## Flow Creation & Modification

Flow creation and modification are handled by dedicated built-in flows — you do NOT need to know node type details.

**Creating a flow:**
1. Call create_flow_conversation with initial_input set to the user's description.
2. The tool BLOCKS until the flow is fully created. Prompts are shown to the user automatically.
3. When the tool returns, summarize the result.

**Modifying a flow:**
1. Use manage_flows action="get" with flow_id and format="json" to fetch the full flow JSON.
2. Use run_flow action="run" with builtin_flow="modify_flow" and initial_input containing the existing flow JSON + the user's modification request.
3. The tool BLOCKS until the modification is complete. Summarize the result.

**Listing / searching / deleting flows:** Use manage_flows with the appropriate action.

## Model Management Rules

- list and get execute immediately.
- create, update, delete return a confirmation prompt with a HIDDEN confirm_key. NEVER display the raw tool response or confirm_key to the user. Instead say: "This is a sensitive operation and requires confirmation." and describe what will be done. Ask "Confirm execution?".
- If the user confirms (yes/ok/confirm/sure/go ahead), call action="confirm" with the confirm_key.
- If the user cancels (no/cancel/nope/never mind), call action="cancel" with the confirm_key.
- API keys are always masked as "****" — never reveal them.
- Supported providers: openai, deepseek-openai, anthropic, gemini, volcengine, openai_compat, claude_compat.
- Model categories: llm, image, voice, video. Default is llm.

## Flow Execution Rules

When a user wants to run a flow:
1. If you know the flow_id, call run_flow action="run" with flow_id (and optional initial_input / form_values).
2. If you only have a name, call action="search" with query first.
3. The tool BLOCKS until the flow completes. If the flow requires user input, prompts are shown to the user automatically via the frontend — you do NOT need to relay them or end your turn. When the flow finishes, the tool returns the final result.
4. Summarize the result for the user.
5. Use action="stop" to cancel a running flow.

## General Behavior

- Before answering, consider which tool to use. Don't guess — search or read when you need real data.
- Use web_search for current events, news, or information beyond your knowledge cutoff.
- Use read_document when the user uploads a file and you need its contents.
- Use execute_command to open apps or run local commands when explicitly asked.
- Keep responses concise and actionable. Use Chinese or English based on the user's language.
- When listing flows or models, present the results in a clear, readable format.`

// ── wsSender (agent → WebSocket bridge) ──

type wsSender struct {
	conn       *websocket.Conn
	runner     *ChatRunner
	onChunk    func(content string, reasoning bool)
	onDone     func()
	onToolCall func(name string, args string)
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
		if s.onToolCall != nil {
			s.onToolCall(event.ToolName, event.ToolInput)
		}
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

// handleAgent runs the agent loop. It blocks until the agent completes.
// Callers should run this in a goroutine so the WebSocket read loop stays
// responsive for flow_user_response and stop messages.
func (r *ChatRunner) handleAgent(conn *websocket.Conn, req WSRequest) {
	cp := r.prepareChat(conn, req)

	// Cancellable context — allows stop/disconnect to abort the agent loop
	// and any blocking tool calls (e.g. run_flow waiting for user input).
	ctx, cancel := context.WithCancel(context.Background())
	r.setAgentCancel(conn, cancel)
	defer r.setAgentCancel(conn, nil)

	sender := &wsSender{conn: conn, runner: r}
	chatID := fmt.Sprintf("%d", cp.sessionID)
	c := agent.NewChat(r.ctx, ctx, cp.sessionID, chatID, cp.modelPath, cp.opts, sender)
	c.SetSystemPrompt(agentSystemPrompt)

	startIter := len(cp.history) / 2
	c.SetIteration(startIter)

	for _, m := range cp.history {
		c.AddMessage(m.Role, m.Content, m.ToolCalls)
	}
	c.AddUserMessage(cp.userMessage)

	var assistantContent strings.Builder
	var assistantToolCalls []common.ToolCall
	sender.onChunk = func(content string, reasoning bool) {
		if !reasoning {
			assistantContent.WriteString(content)
		}
	}
	sender.onToolCall = func(name string, args string) {
		assistantToolCalls = append(assistantToolCalls, common.ToolCall{
			Name:      name,
			Arguments: args,
		})
	}
	sender.onDone = func() {
		if cp.sessionID > 0 && assistantContent.Len() > 0 {
			msg := &entity.ChatMessage{
				SessionId: cp.sessionID,
				Role:      "assistant",
				Content:   assistantContent.String(),
			}
			if len(assistantToolCalls) > 0 {
				if data, err := json.Marshal(assistantToolCalls); err == nil {
					msg.ToolCalls = string(data)
				}
			}
			r.messageModel.Create(msg)
		}
	}
	c.Process()
}
