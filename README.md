# Go AI Agent

A cross-platform desktop AI agent platform with visual flow designer, multi-model support, and streaming chat. Built with **Wails v2** + **React** + **Go**.

## Download

macOS `.app` bundle built from source — see [Build from Source](#build-from-source).

## Features

- **Desktop App** — Native macOS/Windows/Linux window via Wails v2, double-click to launch
- **One-Step Setup** — Desktop mode auto-configures SQLite + admin account, only model API key needed
- **Visual Flow Designer** — Dify-style drag-and-drop DAG editor with 14 node types (LLM, Condition, Loop, Script, etc.)
- **Multi-Model** — OpenAI, Claude, Gemini, DeepSeek, and 28+ providers via unified interface
- **Agent Tool Use** — Built-in tool execution loop with `manage_flows`, `manage_models`, and extensible tool registry
- **Streaming Chat** — WebSocket-based real-time streaming with thinking/reasoning display
- **Web Mode** — `--web` flag runs as a browser-based server (SQLite/MySQL/PostgreSQL, full setup wizard)
- **i18n** — English, 简体中文, 繁體中文, 日本語

## Quick Start

### Desktop App

```bash
# Prerequisites: Go 1.25+, Node 18+, pnpm, Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone and build
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# Dev mode (hot reload)
wails dev

# Build macOS .app bundle
wails build

# Build for specific platforms
wails build -platform darwin/universal   # macOS universal
wails build -platform windows/amd64      # Windows
wails build -platform linux/amd64        # Linux
```

The `.app` bundle is at `build/bin/go-ai-agent.app`. Double-click to launch — first run only asks for your model API key.

### Web Server Mode

```bash
# Build and run as web server (browser access)
go build -o go-ai-agent . && ./go-ai-agent --web

# Frontend dev server
cd view && pnpm dev
```

Open `http://localhost:5173` — first run opens the 3-step setup wizard (database → admin → model).

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
│  ├─ Agent Engine        │            │  ├─ Agent Engine    │
│  └─ Flow Engine         │            │  └─ Flow Engine     │
└─────────────────────────┘            └─────────────────────┘
```

```
main.go
  ├─ Desktop: Wails App → reverse proxy → Go HTTP :19009
  ├─ Web:     Go HTTP :19009 directly
  ├─ Services:  UnifiedChatService (28+ providers, 3 protocols)
  ├─ Runners:   ChatRunner (WebSocket + Agent), FlowRunner (DAG executor)
  └─ REST:      SetupRest, ModelRest, ChatRest, FlowRest
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

## Flow Nodes

**Basic**: Start, End, User Input
**AI**: LLM, Image Generation, Audio Generation, Video Generation
**Logic**: Condition, Transform, Text Split
**Process**: Parallel Batch, Sequential Iterator, Loop, Script

## WebSocket Protocol

Connect to `ws://localhost:19009/ws`. Message types:
- `chat` / `agent` — send to ChatRunner
- `flow_start` / `flow_user_response` / `flow_stop` — flow execution control
- Responses: `chunk`, `tool_call`, `tool_result`, `session_created`, `error`

## Project Structure

```
go-ai-agent/
├── main.go              # Entry point, --web flag, desktop/Web routing
├── app.go               # Wails App struct, asset embedding
├── app_dev.go           # Wails dev mode detection (build tag)
├── app_prod.go          # Wails prod mode detection (build tag)
├── desktop_init.go      # Auto SQLite + admin for desktop mode
├── wails.json           # Wails project config
├── Makefile             # Build targets
├── agent/               # Agent loop, tool registry
├── ai/chat/             # Unified chat service + providers
├── entity/              # DB entity definitions
├── model/               # DB access layer
├── runner/              # ChatRunner, FlowRunner
├── rest/                # REST endpoints (Setup, Chat, Flow, Model)
├── flow/                # Flow engine (DAG executor)
├── build/               # Wails build output (.app bundle)
└── view/                # React frontend
    └── src/
        ├── pages/       # ChatHome, FlowDesigner, ModelManager, SetupWizard
        ├── components/  # assistant-ui adapters, flow editor, common
        ├── stores/      # Zustand state stores
        ├── i18n/        # Locale files (en, zh, zh-TW, ja)
        └── hooks/       # useWebSocket, useChatStream
```

## Makefile

```bash
make desktop-dev          # wails dev (hot reload)
make desktop-build-mac    # macOS universal .app
make desktop-build-win    # Windows .exe
make desktop-build-linux  # Linux binary
```

---

## 简体中文

### 功能特性

- **桌面应用** — 基于 Wails v2 的原生 macOS/Windows/Linux 窗口，双击即用
- **一步配置** — 桌面版自动配置 SQLite + 管理员账号，仅需设置模型 API Key
- **可视化流程设计器** — Dify 风格拖拽式 DAG 编辑器，14 种节点类型
- **多模型支持** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 提供商统一接口
- **Agent 工具调用** — 内置工具执行循环，支持 `manage_flows`、`manage_models`，可扩展
- **流式聊天** — 基于 WebSocket 的实时流式输出，支持思考过程展示
- **Web 模式** — `--web` 参数启动为浏览器服务器（支持 SQLite/MySQL/PostgreSQL，完整设置向导）
- **多语言** — English, 简体中文, 繁體中文, 日本語

### 快速开始

**桌面版：**

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 开发模式（热重载）
wails dev

# 构建 macOS .app
wails build
```

构建产物在 `build/bin/go-ai-agent.app`，双击启动，首次运行仅需配置模型 API Key。

**Web 服务器模式：**

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
```

访问 `http://localhost:5173`，首次运行进入三步设置向导。

---

## 繁體中文

### 功能特性

- **桌面應用** — 基於 Wails v2 的原生 macOS/Windows/Linux 視窗，雙擊即可使用
- **一步配置** — 桌面版自動配置 SQLite + 管理員帳號，僅需設定模型 API Key
- **可視化流程設計器** — Dify 風格拖曳式 DAG 編輯器，14 種節點類型
- **多模型支援** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 供應商統一介面
- **Agent 工具調用** — 內建工具執行迴圈，支援 `manage_flows`、`manage_models`，可擴展
- **串流聊天** — 基於 WebSocket 的即時串流輸出，支援思考過程展示
- **Web 模式** — `--web` 參數啟動為瀏覽器伺服器（支援 SQLite/MySQL/PostgreSQL，完整設定精靈）
- **多語言** — English, 简体中文, 繁體中文, 日本語

### 快速開始

**桌面版：**

```bash
# 安裝 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 開發模式（熱重載）
wails dev

# 構建 macOS .app
wails build
```

構建產物在 `build/bin/go-ai-agent.app`，雙擊啟動，首次執行僅需設定模型 API Key。

**Web 伺服器模式：**

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
```

開啟 `http://localhost:5173`，首次執行進入三步設定精靈。

---

## 日本語

### 機能

- **デスクトップアプリ** — Wails v2によるネイティブ macOS/Windows/Linux ウィンドウ、ダブルクリックで起動
- **ワンステップ設定** — デスクトップ版はSQLite + 管理者アカウントを自動設定、モデルAPIキーのみ必要
- **ビジュアルフローデザイナー** — Difyスタイルのドラッグ＆ドロップDAGエディタ、14種類のノード
- **マルチモデル** — OpenAI、Claude、Gemini、DeepSeekなど28以上のプロバイダーに対応
- **Agentツール実行** — 組み込みツール実行ループ、`manage_flows`、`manage_models`、拡張可能
- **ストリーミングチャット** — WebSocketベースのリアルタイムストリーミング、思考プロセス表示付き
- **Webモード** — `--web` フラグでブラウザサーバーとして起動（SQLite/MySQL/PostgreSQL対応、フルセットアップウィザード）
- **多言語** — English, 简体中文, 繁體中文, 日本語

### クイックスタート

**デスクトップ版：**

```bash
# Wails CLIをインストール
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 開発モード（ホットリロード）
wails dev

# macOS .appをビルド
wails build
```

ビルド成果物は `build/bin/go-ai-agent.app`、ダブルクリックで起動。初回起動時はモデルAPIキーの設定のみ必要です。

**Webサーバーモード：**

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
```

`http://localhost:5173` を開くと、初回実行時に3ステップのセットアップウィザードが表示されます。

## Build from Source

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# Desktop app
make desktop-build-mac      # macOS
make desktop-build-win      # Windows
make desktop-build-linux    # Linux

# Or directly with wails
wails build -platform darwin/universal
```

## License

MIT
