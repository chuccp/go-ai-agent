# Go AI Agent

> 🚧 **Under development** — this project is under active development. Welcome to use Claude Fable 5 to help improve features, thank you very much!

A cross-platform desktop AI agent platform. **Create AI workflows by chatting** — describe what you want in natural language, and the agent designs, builds, and executes the pipeline for you.

Built with **Wails v2** + **React** + **Go**.

[简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

## Why Chat-Created Workflows?

Traditional workflow tools require learning a visual editor — dragging nodes, wiring edges, configuring parameters. With Go AI Agent, you just tell the agent what you need:

> *"Create a flow that fetches the latest AI news, summarizes them with DeepSeek, translates the summary into Japanese, and asks me to review before saving."*

The agent will **understand your intent → propose a node structure → confirm with you → create the flow**. No manual wiring, no config guesswork.

## Features

- **Chat-Created Workflows** — Build AI pipelines through natural language conversation with the `manage_flows` tool
- **Visual Flow Designer** — Drag-and-drop DAG editor with 16 node types including condition, switch, execute, script
- **Script-Based Nodes** — Condition and switch nodes use Starlark (Python dialect) expressions with access to all upstream data
- **Generic Batch Processing** — ForEach and Iterator nodes invoke any function with args, not hardcoded to LLM
- **Desktop App** — Native Windows/macOS/Linux window via Wails v2 with IPC communication
- **One-Step Setup** — Desktop mode auto-configures SQLite + admin account, only model API key needed
- **Shareable Apps** — Export apps as ZIP packages (app.json + meta.json), import with one click
- **Multi-Model** — OpenAI, Claude, Gemini, DeepSeek, and 28+ providers via unified interface
- **Agent Tool Use** — Extensible tool registry with manage_flows, manage_models, execute_command, read_document, web_search
- **Web Mode** — Run as a browser-based server via `cmd/server/main.go`
- **i18n** — English, 简体中文, 繁體中文, 日本語

## Quick Start

### Desktop App (Windows/macOS/Linux)

```bash
# Prerequisites: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# One-click dev mode
dev.bat  # Windows
make desktop-dev  # macOS/Linux

# Or manually
wails dev
```

First run auto-configures SQLite and creates a default admin account (admin/admin). You only need to configure your model API key.

### Web / Server Mode

```bash
go build -o go-ai-agent-server ./cmd/server/
./go-ai-agent-server
```

Open `http://localhost:19009` — first run opens the setup wizard.

## Architecture

```
Desktop Mode                        Web Mode
┌──────────────────────┐            ┌──────────────────────┐
│  Native WebView      │            │  Browser             │
│  ┌────────────────┐  │            └─────────┬────────────┘
│  │  React Frontend │  │                      │ HTTP/WS
│  │  (embedded)     │  │                      │
│  └───────┬────────┘  │            ┌─────────▼────────────┐
└──────────┼───────────┘            │  Go HTTP Server      │
           │ IPC                    │  ├─ REST API         │
┌──────────▼──────────────────────┐ │  ├─ WebSocket        │
│  Go HTTP Server :19009          │ │  ├─ Agent + Tools    │
│  ├─ REST API + CORS             │ │  └─ Flow Engine      │
│  ├─ IPC Events (Wails)          │ └──────────────────────┘
│  ├─ Agent + Tools               │
│  └─ Flow Engine (DAG)           │
└─────────────────────────────────┘
```

**Desktop Mode**: Uses Wails IPC for communication (no WebSocket required)  
**Web Mode**: Uses WebSocket for real-time communication

## Project Structure

```
go-ai-agent/
├── main.go                  # Desktop entry (Wails)
├── cmd/server/main.go       # Web server entry
├── internal/
│   ├── app/                 # Shared setup (config, desktop init, CORS)
│   ├── agent/               # Agent loop and tool registry
│   │   └── tool/            # Tool implementations
│   ├── ai/                  # AI services
│   │   └── chat/            # Unified chat service + 28+ providers
│   ├── entity/              # Database entities (FlowDefinition, AIModel, etc.)
│   ├── model/               # Data access layer
│   ├── rest/                # REST endpoints
│   ├── runner/              # ChatRunner, FlowRunner
│   └── flow/                # Flow engine
│       ├── engine/          # DAG executor, task manager, function registry
│       ├── nodes/           # Node implementations (16 types)
│       └── export/          # ZIP import/export
├── view/                    # React frontend
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, ModelManager, SetupWizard
│       ├── components/      # Shared components (ModelForm, IpcAdapter)
│       ├── stores/          # Zustand state stores
│       └── i18n/            # Locale files (en, zh, zh-TW, ja)
├── wails.json               # Wails project config
├── Makefile                 # Build targets
└── dev.bat                  # One-click desktop dev launcher (Windows)
```

## Flow Engine

**16 node types**: `start`, `end`, `llm`, `skill`, `user_input`, `condition`, `switch`, `transform`, `split`, `for_each`, `iterator`, `loop`, `script`, `execute`, `image_gen`, `audio_gen`, `video_gen`

**Skill Node**: Direct prompt execution with model selection
```json
{ "prompt": "{{start.output}}", "model": "1.default" }
```

**Script-Based Nodes** use Starlark (Python dialect):
```python
# Condition: returns bool → "yes"/"no" branch
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# Switch: returns string → routes to matching source_handle
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

**Generic Batch Processing** — ForEach and Iterator invoke any registered function:
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```
ForEach runs in parallel, Iterator runs sequentially (skips failures).

**Execute Node** runs local shell commands with configurable timeout (`0` = no limit).

**App Export** uses ZIP format (`app.json` + `meta.json`).

## Communication Protocol

### Desktop Mode (IPC)
- Uses Wails Events for real-time communication
- Event pattern: `chat:{sessionId}:{type}` (e.g., `chat:5:chunk`)
- Event types: `chunk`, `tool_call`, `tool_result`, `error`, `session_created`

### Web Mode (WebSocket)
- Connect to `ws://localhost:19009/ws/chat`
- Message types:
  - `chat` / `agent` — sends to ChatRunner
  - `flow_start` / `flow_user_response` / `flow_stop` — flow execution control
  - Responses: `chunk`, `tool_call`, `tool_result`, `error`, `session_created`

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop Shell | Wails v2 (system WebView) |
| Backend | Go + go-web-frame + CORS middleware |
| Frontend | React 18 + TypeScript + Vite |
| Flow Editor | reactflow + Zustand |
| Chat UI | @assistant-ui/react |
| i18n | react-i18next |
| Database | SQLite (desktop) / MySQL / PostgreSQL (web) |
| Communication | IPC (desktop) / WebSocket (web) |

## License

MIT
