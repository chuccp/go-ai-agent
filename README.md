# go-ai-agent

A self-hosted AI agent platform with visual flow designer, multi-model support, and streaming chat.

## Features

- **Visual Flow Designer** — Dify-style drag-and-drop DAG editor with 14 node types (LLM, Condition, Loop, Script, etc.)
- **Multi-Model** — OpenAI, Claude, Gemini, DeepSeek, and 28+ providers via unified interface
- **Agent Tool Use** — Built-in tool execution loop with `manage_flows`, `manage_models`, and extensible tool registry
- **Streaming Chat** — WebSocket-based real-time streaming with thinking/reasoning display
- **Multi-Database** — SQLite, MySQL, PostgreSQL support
- **i18n** — English, 简体中文, 繁體中文, 日本語
- **First-Run Wizard** — Guided setup for database, admin account, and base model

## Quick Start

```bash
# Clone and build
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# Backend (requires Go 1.25+)
go build -o go-ai-agent . && ./go-ai-agent

# Frontend (requires Node 18+)
cd view && pnpm install && pnpm dev
```

Open `http://localhost:5173` — first run opens the setup wizard.

## Architecture

```
main.go → go-web-frame Builder
  ├─ Services:  UnifiedChatService (28+ providers, 3 protocols)
  ├─ Runners:   ChatRunner (WebSocket + Agent), FlowRunner (DAG executor)
  ├─ REST:      Api, SetupRest, ModelRest, ChatRest, FlowRest
  └─ Agent:     Tool loop (max 10 iterations), extensible tool registry
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go + go-web-frame + Gorilla WebSocket |
| Frontend | React 18 + TypeScript + Vite |
| Flow Editor | reactflow + Zustand |
| Chat UI | @assistant-ui/react |
| i18n | react-i18next |
| Database | SQLite / MySQL / PostgreSQL |

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
├── main.go              # Entry point, config loading
├── agent/               # Agent loop, tool registry
├── ai/chat/             # Unified chat service + providers
├── entity/              # DB models
├── model/               # DB access layer
├── runner/              # ChatRunner, FlowRunner
├── rest/                # REST endpoints
├── flow/                # Flow engine (DAG executor)
└── view/                # React frontend
    └── src/
        ├── pages/       # ChatHome, FlowDesigner, ModelManager, SetupWizard
        ├── components/  # assistant-ui adapters, flow editor, common
        ├── stores/      # Zustand state stores
        ├── i18n/        # Locale files (en, zh, zh-TW, ja)
        └── hooks/       # useWebSocket, useChatStream
```

---

## 简体中文

### 功能特性

- **可视化流程设计器** — Dify 风格拖拽式 DAG 编辑器，14 种节点类型
- **多模型支持** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 提供商统一接口
- **Agent 工具调用** — 内置工具执行循环，支持 `manage_flows`、`manage_models`，可扩展
- **流式聊天** — 基于 WebSocket 的实时流式输出，支持思考过程展示
- **多数据库** — 支持 SQLite、MySQL、PostgreSQL
- **多语言** — English, 简体中文, 繁體中文, 日本語
- **首次运行向导** — 引导式配置数据库、管理员账号和基础模型

### 快速开始

```bash
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 后端 (Go 1.25+)
go build -o go-ai-agent . && ./go-ai-agent

# 前端 (Node 18+)
cd view && pnpm install && pnpm dev
```

访问 `http://localhost:5173`，首次运行会进入设置向导。

---

## 繁體中文

### 功能特性

- **可視化流程設計器** — Dify 風格拖曳式 DAG 編輯器，14 種節點類型
- **多模型支援** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 供應商統一介面
- **Agent 工具調用** — 內建工具執行迴圈，支援 `manage_flows`、`manage_models`，可擴展
- **串流聊天** — 基於 WebSocket 的即時串流輸出，支援思考過程展示
- **多資料庫** — 支援 SQLite、MySQL、PostgreSQL
- **多語言** — English, 简体中文, 繁體中文, 日本語
- **首次執行精靈** — 引導式設定資料庫、管理員帳號和基礎模型

### 快速開始

```bash
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 後端 (Go 1.25+)
go build -o go-ai-agent . && ./go-ai-agent

# 前端 (Node 18+)
cd view && pnpm install && pnpm dev
```

開啟 `http://localhost:5173`，首次執行會進入設定精靈。

---

## 日本語

### 機能

- **ビジュアルフローデザイナー** — Difyスタイルのドラッグ＆ドロップDAGエディタ、14種類のノード
- **マルチモデル** — OpenAI、Claude、Gemini、DeepSeekなど28以上のプロバイダーに対応
- **Agentツール実行** — 組み込みツール実行ループ、`manage_flows`、`manage_models`、拡張可能
- **ストリーミングチャット** — WebSocketベースのリアルタイムストリーミング、思考プロセス表示付き
- **マルチデータベース** — SQLite、MySQL、PostgreSQL対応
- **多言語** — English, 简体中文, 繁體中文, 日本語
- **初回セットアップウィザード** — データベース、管理者アカウント、ベースモデルのガイド付き設定

### クイックスタート

```bash
git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# バックエンド (Go 1.25+)
go build -o go-ai-agent . && ./go-ai-agent

# フロントエンド (Node 18+)
cd view && pnpm install && pnpm dev
```

`http://localhost:5173` を開くと、初回実行時にセットアップウィザードが表示されます。

## License

MIT
