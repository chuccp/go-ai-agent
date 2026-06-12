# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Desktop dev (Windows, Wails + Vite)
dev.bat                        # one-click: kill old procs, start wails dev
wails dev                      # manual start from project root

# Web/Server mode (port 19009, no GUI)
go build -o go-ai-agent.exe ./cmd/server/   # pure server entry
./go-ai-agent.exe

# Frontend (port 5173, proxies /api → 19009)
cd view && pnpm dev
cd view && pnpm build        # production build into view/dist/
```

The Go module depends on a local replacement: `go-web-frame` at `../go-web-frame`.

## Multi-Entry Architecture

```
main.go              → Desktop entry (Wails, embeds view/dist/)
cmd/server/main.go   → Pure web server entry (no Wails dependency)
internal/app/        → Shared setup: Create(webMode), CreateDesktop(), DesktopInitService
```

Desktop mode auto-configures SQLite at `./data/go-ai-agent.db` and creates a default admin (admin/admin). It also sets `system.desktop=true` in the config and registers CORS middleware for cross-origin requests from the Wails webview.

The frontend detects Wails dev mode (`wails.localhost` hostname) and points API_BASE to `http://localhost:19009` directly, relying on the Go server's CORS headers. In web mode, API_BASE is empty (relative paths through Vite proxy).

## Architecture

```
main.go / cmd/server/main.go → go-web-frame Builder
  ├─ Services:  UnifiedChatService, DesktopInitService (desktop only), CorsFilter
  ├─ Models:    ChatSession, ChatMessage, AIModel, Flow*, AdminUser
  ├─ Runners:   ChatRunner (WebSocket + agent), FlowRunner (DAG executor)
  └─ REST:      Api, SetupRest, ModelRest, ChatRest, FlowRest
```

**Key design decisions:**

- **First-run mode**: `loadOrCreateConfig()` checks `system.init`. If false, registers `SetupRest` endpoints. After setup completes, writes `application.yml` with `system.init: true`. On next restart, SetupRest is not registered — only `Api.getSetupStatus()` remains for status checks.
- **Model config**: AI model credentials live in the DB (`ai_models` table), NOT in `application.yml`. On startup, `ChatRunner.Init()` loads all `AIModel` records and calls `ConfigureProvider()`.
- **Desktop auto-init**: `DesktopInitService` auto-creates SQLite DB + default admin when `system.desktop=true` and `system.init=false`. The frontend setup wizard then shows only the model configuration step (DB and admin steps are pre-completed).
- **CORS**: `go-web-frame/component/cors` filter is always registered for Wails desktop dev compatibility.

## Frontend (React + Vite + Zustand)

| Path | Component |
|------|-----------|
| `/` | ChatHome (chat + flow selector) |
| `/designer` | FlowDesigner (list view) |
| `/designer/:id` | FlowDesigner (canvas editor, @vue-flow) |
| `/models` | ModelManager |
| `/setup` | SetupWizard (first-run, desktop skips DB/admin steps) |

Hash-based routing via react-router-dom. State management with Zustand (`flowStore.ts`, `setupStore.ts`, `modelStore.ts`). Shared `ModelForm` component used by both `SetupWizard` and `ModelManager`.

Vite config proxies `/api` → `localhost:19009` and `/ws` → `ws://localhost:19009` for web dev mode. In Wails dev mode, `constants.ts` detects the `wails.localhost` hostname and returns `http://localhost:19009` as `API_BASE`.

## Flow Engine

**Node types** — registered in `FlowRunner.Init()`:
- Basic: `start`, `end`, `user_input`
- AI: `llm`, `image_gen`, `audio_gen`, `video_gen` (gen nodes are stubs)
- Logic: `condition` (Starlark bool), `switch` (Starlark string), `loop`
- Process: `for_each` (parallel), `iterator` (sequential), `split`, `transform`, `script`, `execute`

**Script-based nodes** (`condition`, `switch`, `script`) use Starlark with access to `ctx["node_label"]["field"]` for all upstream outputs, plus `json_parse(s)` and `split(s, sep)` builtins. Shared Starlark env helpers live in `flow/nodes/starlark_env.go`.

**Generic batch nodes** (`for_each`, `iterator`): config is `{items_key, function, args}`. They invoke any registered function (default `"llm"`), with `{{item.field}}` placeholder support in args. For_each runs items in parallel, iterator runs sequentially (skips failures).

**Execute node**: runs local shell commands with configurable timeout (`0` = no timeout, default 30s). Failures don't block the flow.

**DAG execution** via `flow/engine/` with parallel task manager (max 4 concurrent). The engine records router node (condition/switch) NextNode values for branch-skipping logic.

## Flow Export/Import

ZIP-based format (`flow/export/package.go`):
- `meta.json` — type/version/kind/timestamp
- `flow.json` — name, description, category, nodes, edges (IDs stripped)
- `skills/`, `resources/` directories (placeholder for future)

Endpoints: `GET /api/flows/:id/export` (download ZIP) and `POST /api/flows/import` (upload ZIP).

## Chat System

```
UnifiedChatService → map[string]ChatProvider
  ├─ openai.*   (OpenAI protocol)
  ├─ claude.*   (Anthropic Messages protocol)
  └─ native.*   (gemini, volcengine — custom protocols)
```

## Agent Loop

```
agent.Chat.Process()
  for iteration < MaxIterations (10):
    resp = svc.ChatWithTools(path, history, text, opts)
    if ToolCalls → execute tools → append results → continue
    else → emit text → done
```

Tools registered via `init()` in `agent/tool/*.go` using `tool.Register(Executor)`. The `manage_flows` tool guides the LLM through: understand → design → confirm → create.

## Code Patterns

- **Service/model lookup**: `core.GetService[*chat.UnifiedChatService](ctx)` and `core.GetModel[*model.AIModelModel](ctx)` — typed generics from go-web-frame.
- **REST responses**: `web.Data(v)` → `{"code":200,"msg":"ok","data":v}`. `web.Ok(msg)` for success messages.
- **Config persistence**: `cfg.Put("key", value)` writes to viper runtime config. Only system/db/server/flow sections are persisted — chat config stays in the DB.
- **Frontend state**: Zustand stores in `view/src/stores/`. `useFlowStore` for flows, `useSetupStore` for setup wizard, `useModelStore` for AI models.
- **i18n**: react-i18next with 4 locales (en, zh, zh-TW, ja) in `view/src/i18n/locales/`.
- **No auth**: No authentication middleware yet (planned).
