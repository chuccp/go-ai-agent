# Go AI Agent

> 🚧 **Under Development** — This project is under active development. Contributions are welcome!

A cross-platform desktop AI agent platform. **Chat to create apps** — describe what you want in natural language, and the agent designs, builds, and executes the flow for you.

Built with **Wails v2** + **React** + **Go**.

[简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

## Core Concept: App = Flow

In Go AI Agent, **an App is a Flow**. There's no separate "package" or "skill" management — you create an app, design its flow visually or via chat, and run it. Everything is unified under a single concept.

- **App**: A complete, self-contained workflow with nodes, edges, and configurations
- **Flow**: The visual representation of an app's logic
- **Skill Node**: A node that executes a prompt directly (no external skill management needed)

## Features

- **Chat-Created Apps** — Build complete workflows through natural language conversation
- **Visual Flow Designer** — Drag-and-drop DAG editor with 17 node types
- **17 Node Types**: start, end, llm, skill, user_input, condition, switch, transform, split, for_each, iterator, loop, script, execute, image_gen, audio_gen, video_gen
- **Skill Nodes** — Execute prompts directly within the flow (no external skill management)
- **Script-Based Logic** — Condition and switch nodes use Starlark (Python dialect) for complex branching
- **Batch Processing** — for_each (parallel) and iterator (sequential) nodes for array processing
- **Desktop App** — Native Windows/macOS/Linux via Wails v2 with IPC communication
- **Web Mode** — Run as a browser-based server with WebSocket communication
- **One-Step Setup** — Desktop mode auto-configures SQLite + admin account
- **App Export** — Export apps as ZIP packages, import with one click
- **Multi-Model** — OpenAI, Claude, Gemini, DeepSeek, and 28+ providers via unified interface
- **Agent Tool Use** — Extensible tool registry: manage_flows, manage_models, execute_command, read_document, web_search
- **i18n** — English, 简体中文, 繁體中文, 日本語

## Quick Start

### Desktop App

```bash
# Prerequisites: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# Development mode (hot reload)
make desktop-dev  # macOS/Linux
dev.bat           # Windows

# Build production app
make desktop-build  # macOS/Linux
wails build         # manual
```

First run auto-configures SQLite and creates a default admin account (admin/admin). You only need to configure your model API key.

### Web Server Mode

```bash
make server-build  # macOS/Linux
go build -o go-ai-agent-server.exe ./cmd/server/  # Windows

./go-ai-agent-server
```

Open `http://localhost:19009` — first run opens the setup wizard.

## Architecture

### Desktop Mode (IPC)
```
┌─────────────────────────────────┐
│  Native WebView (Wails v2)      │
│  ┌───────────────────────────┐  │
│  │  React Frontend           │  │
│  │  - ChatHome               │  │
│  │  - FlowDesigner           │  │
│  │  - ModelManager           │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ Wails IPC (Events)
┌─────────────┼───────────────────┐
│  Go Backend :19009              │
│  ├─ REST API (/api/*)           │
│  ├─ IPC Event Bus               │
│  ├─ Agent + Tools               │
│  └─ Flow Engine (DAG Executor)  │
└─────────────────────────────────┘
```

### Web Mode (WebSocket)
```
┌─────────────────────────────────┐
│  Browser                        │
│  ┌───────────────────────────┐  │
│  │  React Frontend           │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ WebSocket (/ws/chat)
              │ HTTP (/api/*)
┌─────────────┼───────────────────┐
│  Go Backend :19009              │
│  ├─ REST API                    │
│  ├─ WebSocket Server            │
│  ├─ Agent + Tools               │
│  └─ Flow Engine                 │
└─────────────────────────────────┘
```

**Communication Protocol:**
- Desktop: Wails IPC events (e.g., `chat:{sessionId}:chunk`)
- Web: WebSocket messages (JSON format)

## Node Types

### Basic Nodes
- **start**: Flow entry point
- **end**: Flow exit point
- **user_input**: Wait for user input or confirmation

### AI Nodes
- **llm**: Call LLM with prompt and system message
- **skill**: Execute prompt directly (simplified LLM node)
- **image_gen**: Generate images via AI models
- **audio_gen**: Generate audio/speech via AI models
- **video_gen**: Generate videos via AI models

### Logic Nodes
- **condition**: if/else branching (Starlark boolean expression)
- **switch**: Multi-way branching (Starlark string expression)
- **loop**: Repeat execution until condition is met

### Data Processing Nodes
- **transform**: Go template-based data transformation
- **split**: Split text by delimiter into JSON array
- **for_each**: Parallel processing of array items
- **iterator**: Sequential processing of array items

### Execution Nodes
- **script**: Starlark Python custom code
- **execute**: Run local shell commands

## Script-Based Nodes

**Condition** and **switch** nodes use Starlark (Python dialect):

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

## Batch Processing

**for_each** runs in parallel:
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

**iterator** runs sequentially (skips failures):
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

## Skill Nodes

Skill nodes execute prompts directly — no external skill management:

```json
{
  "prompt": "Summarize the following text:\n\n{{llm.output}}",
  "model": "1.default"
}
```

Skill nodes are essentially simplified LLM nodes for quick prompt execution within a flow.

## App Export Format

Apps are exported as ZIP packages containing:
- `meta.json`: App metadata (name, icon, description)
- `app.json`: Flow definition with nodes and edges
- `resources/`: Additional files (if any)

```bash
# Export: App → ZIP file
# Import: ZIP file → App
```

## Project Structure

```
go-ai-agent/
├── main.go                  # Desktop entry (Wails)
├── cmd/server/main.go       # Web server entry
├── internal/
│   ├── agent/               # Agent loop and tool registry
│   │   └── tool/            # Tool implementations
│   ├── ai/                  # AI services
│   │   └── chat/            # Unified chat service + 28+ providers
│   ├── app/                 # Application setup and configuration
│   ├── config/              # Configuration management
│   ├── entity/              # Database entities (FlowDefinition, AIModel, etc.)
│   ├── flow/                # Flow engine
│   │   ├── engine/          # DAG executor, task manager, function registry
│   │   ├── nodes/           # 17 node type implementations
│   │   └── export/          # App export/import (ZIP format)
│   ├── model/               # Data access layer
│   ├── rest/                # REST API endpoints
│   ├── runner/              # ChatRunner, FlowRunner
│   ├── service/             # Business logic services
│   └── util/                # Utilities
├── view/                    # React frontend
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, FlowRunner, ModelManager, SetupWizard
│       ├── components/      # Shared components (ModelForm, IpcAdapter, etc.)
│       ├── stores/          # Zustand state stores
│       └── i18n/            # Locale files (en, zh, zh-TW, ja)
├── wails.json               # Wails project config
├── Makefile                 # Build targets
└── dev.bat                  # One-click desktop dev launcher (Windows)
```

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
