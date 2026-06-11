# Go AI Agent

A cross-platform desktop AI agent platform. **Create AI workflows by chatting** — describe what you want in natural language, and the agent designs, builds, and executes the pipeline for you.

Built with **Wails v2** + **React** + **Go**.

[简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

## Why Chat-Created Workflows?

Traditional workflow tools require learning a visual editor — dragging nodes, wiring edges, configuring parameters. With Go AI Agent, you just tell the agent what you need:

> *"Create a flow that fetches the latest AI news, summarizes them with DeepSeek, translates the summary into Japanese, and asks me to review before saving."*

The agent will **understand your intent → propose a node structure → confirm with you → create the flow**. No manual wiring, no config guesswork. You can refine iteratively through conversation — "change the model to GPT-5", "add a sentiment analysis step before summarization" — and the agent updates the flow immediately.

**Benefits over manual editors:**

- **Zero learning curve** — no need to learn node types, connection rules, or config schemas
- **Natural iteration** — refine flows through conversation, like talking to a colleague
- **Design guidance** — the agent suggests best practices (e.g., "this flow should have a user confirmation step before sending")
- **Fast prototyping** — go from idea to working pipeline in under a minute
- **Full control** — visual editor still available for manual fine-tuning anytime

## Features

- **Chat-Created Workflows** — Build AI pipelines through natural language conversation with the `manage_flows` tool
- **Visual Flow Designer** — Dify-style drag-and-drop DAG editor with 14 node types, for manual editing when needed
- **Desktop App** — Native macOS/Windows/Linux window via Wails v2, double-click to launch
- **One-Step Setup** — Desktop mode auto-configures SQLite + admin account, only model API key needed
- **Multi-Model** — OpenAI, Claude, Gemini, DeepSeek, and 28+ providers via unified interface
- **Agent Tool Use** — Built-in tool execution loop with extensible tool registry
- **Streaming Chat** — WebSocket-based real-time streaming with thinking/reasoning display
- **Web Mode** — `--web` flag runs as a browser-based server (SQLite/MySQL/PostgreSQL)
- **i18n** — English, 简体中文, 繁體中文, 日本語

## Quick Start

### Desktop App

```bash
# Prerequisites: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# Dev mode (hot reload)
wails dev

# Build macOS .app
wails build
```

The `.app` bundle is at `build/bin/go-ai-agent.app`. Double-click to launch — first run only asks for your model API key.

### Web Server Mode

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
cd view && pnpm dev
```

Open `http://localhost:5173` — first run opens the setup wizard.

## How It Works

```
You: "Create a content review flow"
         │
         ▼
┌─────────────────────────────────┐
│  AI Agent (manage_flows tool)   │
│                                 │
│  1. UNDERSTAND — ask clarifying │
│     questions about the flow    │
│  2. DESIGN — propose node       │
│     structure in plain language │
│  3. CONFIRM — wait for your     │
│     explicit approval           │
│  4. CREATE — build the flow     │
│     with correct nodes & edges  │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Visual Flow (auto-created)     │
│  ┌───┐   ┌──────┐   ┌───┐     │
│  │Start│──▶│ LLM  │──▶│End│     │
│  └───┘   └──────┘   └───┘     │
│                                 │
│  Edit manually anytime in the   │
│  drag-and-drop designer         │
└─────────────────────────────────┘
```

## Architecture

```
Desktop Mode (default)              Web Mode (--web)
┌──────────────────────┐            ┌──────────────────────┐
│  Native WebView      │            │  Browser             │
│  ┌────────────────┐  │            │  http://localhost:   │
│  │  React Frontend │  │            │    5173 (dev)        │
│  │  (embedded)     │  │            │    19009 (prod)      │
│  └───────┬────────┘  │            └─────────┬────────────┘
│          │ reverse    │                      │ HTTP/WS
│          │ proxy      │                      │
┌─┴─────────▼──────────┴─┐            ┌────────▼────────────┐
│  Go HTTP Server :19009 │            │  Go HTTP Server     │
│  ├─ REST API            │            │  ├─ REST API        │
│  ├─ WebSocket           │            │  ├─ WebSocket       │
│  ├─ Agent + Tools       │            │  ├─ Agent + Tools   │
│  └─ Flow Engine (DAG)   │            │  └─ Flow Engine     │
└─────────────────────────┘            └─────────────────────┘
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Desktop Shell | Wails v2 (system WebView) |
| Backend | Go + go-web-frame + Gorilla WebSocket |
| Frontend | React 18 + TypeScript + Vite |
| Flow Editor | reactflow + Zustand |
| Chat UI | @assistant-ui/react |
| i18n | react-i18next |
| Database | SQLite (desktop) / MySQL / PostgreSQL (web) |

## Project Structure

```
go-ai-agent/
├── main.go              # Entry point, --web flag, desktop/web routing
├── app.go               # Wails App struct, asset embedding
├── desktop_init.go      # Auto SQLite + admin for desktop mode
├── wails.json           # Wails project config
├── Makefile             # Build targets
├── agent/               # Agent loop, tool registry (manage_flows, manage_models, ...)
├── ai/chat/             # Unified chat service + 28+ providers
├── runner/              # ChatRunner (WebSocket + Agent), FlowRunner (DAG executor)
├── rest/                # REST endpoints
├── flow/                # Flow engine (DAG executor, 14 node types)
└── view/                # React frontend
    └── src/
        ├── pages/       # ChatHome, FlowDesigner, ModelManager, SetupWizard
        ├── components/  # Chat UI, flow editor, common components
        ├── stores/      # Zustand state stores
        └── i18n/        # Locale files (en, zh, zh-TW, ja)
```

## Makefile

```bash
make desktop-dev          # wails dev (hot reload)
make desktop-build-mac    # macOS universal .app
make desktop-build-win    # Windows .exe
make desktop-build-linux  # Linux binary
```

## WebSocket Protocol

Connect to `ws://localhost:19009/ws`. Message types:
- `chat` / `agent` — send to ChatRunner (agent loop with tool use)
- `flow_start` / `flow_user_response` / `flow_stop` — flow execution control
- Responses: `chunk`, `tool_call`, `tool_result`, `session_created`, `error`

## License

MIT
