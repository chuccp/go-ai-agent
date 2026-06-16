package runner

import (
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
| create_flow_conversation | Start the conversational assistant to create a flow by chatting |
| manage_flows | Create, list, search, get, update, delete flows manually |
| manage_models | List, get, create, update, delete AI model credentials |
| run_flow | Search flows by name, run/respond/status/stop flow executions |
| web_search | Search the internet for real-time information |
| read_document | Extract text from uploaded files (TXT, DOCX, XLSX, PDF) |
| execute_command | Run local shell commands (open apps, manage files, etc.) |

## Flow Creation Rules

When the user wants to create a new flow, ALWAYS call create_flow_conversation first. This is the preferred method. Pass the user's description as initial_input.

1. **START** — Call create_flow_conversation with initial_input set to the user's description.
2. **RELAY** — If the tool result contains waiting_prompt, show that exact question to the user and end your turn.
3. **RESPOND** — When the user answers, call run_flow action="respond" with execution_id and their answer.
4. **COMPLETE** — The assistant will generate and save the flow automatically. Summarize the result for the user.

Only use manage_flows create if the user explicitly says they want manual control over every node and edge, or if create_flow_conversation cannot satisfy the request.

### Node Types Reference

Every node type has REQUIRED config fields that MUST be filled in:

| Node | Required Config | Optional Config |
|------|----------------|-----------------|
| start | (none) | — |
| end | (none) | — |
| llm | prompt, model | system, temperature(0-2), top_p(0-1), max_tokens, thinking_level(off\|low\|medium\|high\|max), output_format_type |
| user_input | prompt | confirm_only(bool) |
| split | source_key, delimiter(paragraph\|line\|，\|。) | — |
| condition | script (Starlark, must assign bool to 'result') | — |
| switch | script (Starlark, must assign string to 'result') | — |
| transform | template (supports {{NodeLabel.field}}) | — |
| for_each | items_key | function(default "llm"), args (supports {{item.field}}) |
| iterator | items_key | function(default "llm"), args (supports {{item.field}}) |
| loop | max_iterations | break_field, break_operator(==\|!=\|>\|<\|>=\|<=\|contains), break_value |
| script | script (Starlark code) | — |
| execute | command (shell command, supports {{NodeLabel.output}}) | timeout(seconds, 0=no timeout, default 30) |
| skill | prompt | model (falls back to system default) |
| image_gen | prompt | model |
| audio_gen | text, model | voice |
| video_gen | prompt | model, duration |

- edges use source_index/target_index (0-based). source_handle: "output" (default), "yes"/"no" (condition), or case values (switch).
- Starlark scripts access upstream data via ctx["node_label"]["field"]. Built-in helpers: json_parse(s), split(s, sep).
- Never create a node with empty required fields. Ask the user for any missing values.

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
3. After calling action="run" or action="respond", the tool result may contain a waiting_prompt. If waiting_prompt is present, you MUST stop calling tools, relay that exact question to the user, and end your turn. Do NOT repeatedly call action="status".
4. When the user replies to a waiting prompt, call run_flow action="respond" with execution_id and the user's answer.
5. If a respond result has no waiting_prompt and status is completed, summarize the result for the user.
6. Use action="stop" to cancel a running flow.

## General Behavior

- Before answering, consider which tool to use. Don't guess — search or read when you need real data.
- Use web_search for current events, news, or information beyond your knowledge cutoff.
- Use read_document when the user uploads a file and you need its contents.
- Use execute_command to open apps or run local commands when explicitly asked.
- Keep responses concise and actionable. Use Chinese or English based on the user's language.
- When listing flows or models, present the results in a clear, readable format.`

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
	c := agent.NewChat(r.ctx, chatID, cp.modelPath, cp.opts, sender)
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
