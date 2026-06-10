# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Backend (port 19009)
go build -o go-ai-agent . && ./go-ai-agent

# Frontend dev server (port 5173, proxies /api → 19009)
cd view && pnpm dev

# Frontend type-check
cd view && npx vue-tsc --noEmit
```

The Go module depends on a local replacement: `go-web-frame` at `../go-web-frame`. Make sure that repo is cloned alongside this one.

## Architecture

```
main.go → go-web-frame Builder
  ├─ Services:  UnifiedChatService  (28 providers, 3 protocols)
  ├─ Models:    ChatSession, ChatMessage, AIModel, Flow*, AdminUser
  ├─ Runners:   ChatRunner (WebSocket chat + agent), FlowRunner (DAG executor)
  └─ REST:      Api, SetupRest, ModelRest, ChatRest, FlowRest
```

**Key design decisions:**

- **First-run mode**: `loadOrCreateConfig()` in main.go checks `system.init`. If false, registers `SetupRest` endpoints (`/api/setup/*`). After setup completes, writes `application.yml` with `system.init: true` and on next restart the setup endpoints are not registered.
- **Model config flow**: AI model credentials live in the DB (`ai_models` table), NOT in `application.yml`. On startup, `ChatRunner.Init()` loads all `AIModel` records and calls `ConfigureProvider()` on the chat service. The `chat:` section was deliberately removed from `application.yml`.
- **Provider defaults for setup wizard**: `GetGroupedProviderInfo()` reads from the `ProviderDefaults` maps in each provider package (openai/claude/gemini/volcengine) — static data, no service dependency. The setup wizard calls `GET /api/setup/providers` to get a two-level `{openai: {provider: {model, baseUrl}}, claude: {...}, native: {...}}` structure for auto-fill.

## Chat System

```
UnifiedChatService
  └─ map[string]ChatProvider   (registered by NewDefaultChatService())
       ├─ openai.*   (~11 providers, OpenAI Chat Completions protocol)
       ├─ claude.*   (~11 providers, Anthropic Messages protocol)
       └─ native.*   (gemini, volcengine — custom protocols)

Usage: service.GetChatService("openai.gpt-4o") → ChatService
       service.ChatStream("deepseek.default", text, handler, opts)
```

- **Config keys**: `chat.{name}.apiKey`, `chat.{name}.model`, `chat.{name}.baseUrl`
- **`ConfigureProvider(name, apiKey, model, baseURL)`** sets these keys in viper and re-initializes the provider. Called at startup from DB models.
- Providers that fail `Init()` (no API key) are kept registered for metadata queries — critical for the setup wizard flow.
- Interfaces and shared types (`ChatProvider`, `ChatService`, `ChatMessage`, `LLMOptions`, etc.) live in `chat/common/` to avoid circular imports.

## Agent Loop

```
agent.Chat.Process()
  for iteration < MaxIterations (10):
    resp = svc.ChatWithTools(path, history, text, opts)
    if resp has ToolCalls:
      execute tools → append results to history → continue loop
    else:
      emit text response → done
```

- **Tools**: Registered via `init()` in `agent/tool/*.go` using `tool.Register(Executor)`. The `Executor` interface is `Definition() + Execute(Call) (string, error)`.
- **`manage_flows`** tool lets the agent create/edit/delete flows through conversation. The tool description instructs the LLM to: understand → design → confirm → create.
- Tools are injected into LLM requests via `c.opts.SetTools(tool.List())` at the start of `Process()`.

## Flow Engine

- **Node types** (registered in `FlowRunner.Init()`): start, end, llm, user_input, split, condition, transform, for_each, iterator, loop, script, image_gen, audio_gen, video_gen
- Image/audio/video gen nodes are registered but have stub implementations.
- DAG execution via `flow/engine/` with parallel task manager (max 4 concurrent).
- Runtime state stored in `FlowExecution` entity (status, current_node_id, context JSON).

## WebSocket Protocol

ChatHome.vue connects to `ws://localhost:19009/ws`. Message types:
- `chat` / `agent` — sends to ChatRunner
- `flow_start` / `flow_user_response` / `flow_stop` — flow execution control
- Responses: `chunk` (streaming text), `tool_call`, `tool_result`, `error`, `session_created`

## Database

First-run wizard supports SQLite (default), MySQL, PostgreSQL. Framework auto-migrates all model tables on startup.

## Frontend Routes

| Path | Component |
|------|-----------|
| `/` | ChatHome (chat + flow selector) |
| `/designer` | FlowDesigner (list view) |
| `/designer/:id` | FlowDesigner (canvas editor, uses @vue-flow) |
| `/models` | ModelManager |
| `/setup` | SetupWizard (first-run only) |

Uses hash-based routing, Pinia for state, `@vue-flow/core` for the flow canvas.

## Code Patterns

- **Service/model lookup**: `core.GetService[*chat.UnifiedChatService](ctx)` and `core.GetModel[*model.AIModelModel](ctx)` — typed generic helpers from go-web-frame.
- **REST responses**: Wrap with `web.Data(v)` → `{"code":200, "msg":"ok", "data":v}`. Use `web.Ok(msg)` for success messages.
- **Config persistence**: `cfg.Put("key", value)` writes to viper runtime config. `WriteAppConfig()` marshals to YAML and writes `application.yml`. Only system/db/server/flow sections are persisted — chat config stays in the DB.
- **No auth**: There is no authentication middleware yet. This is planned after the core flow is complete.
