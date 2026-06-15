# Go AI Agent

> 🚧 **開發中** — 專案正在活躍開發，歡迎使用 Claude Fable 5 來幫助完善功能，非常感謝！

跨平台桌面 AI 智能體平台。**透過聊天建立 AI 工作流程** — 用自然語言描述你的需求，AI 智能體為你設計、構建並執行流水線。

基於 **Wails v2** + **React** + **Go** 構建。

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

## 為何用聊天建立工作流程？

傳統的工作流程工具需要學習視覺化編輯器 — 拖曳節點、連接邊線、配置參數。使用 Go AI Agent，你只需要告訴它你想要什麼：

> *"幫我建立一個流程：擷取最新的 AI 新聞，用 DeepSeek 進行摘要，把摘要翻譯成日語，讓我審核後再儲存。"*

智能體會**理解你的意圖 → 提出節點結構方案 → 等待你確認 → 自動建立流程**。不需要手動連線，不需要猜測配置。你可以透過對話逐步優化 — "把模型換成 GPT-5"、"在摘要之前加一步情感分析" — 智能體立即更新流程。

**相較手動編輯器的優勢：**

- **零學習成本** — 不需要學習節點類型、連線規則、配置參數
- **自然迭代** — 透過對話優化流程，就像和同事討論一樣
- **設計指導** — 智能體會建議最佳實踐（如「這個流程在發送前應加一個使用者確認步驟」）
- **快速原型** — 從想法到可執行的流水線，不到一分鐘
- **完全可控** — 隨時可以在視覺化編輯器中手動微調

## 功能特性

- **聊天建立工作流程** — 透過自然語言對話，使用 `manage_flows` 工具構建 AI 流水線
- **視覺化流程設計器** — Dify 風格拖曳式 DAG 編輯器，16 種節點類型，需要時可手動編輯
- **腳本節點** — 條件和切換節點使用 Starlark（Python 方言）表達式，可存取所有上游資料
- **通用批次處理** — ForEach 和 Iterator 節點呼叫任意函數，不侷限於 LLM
- **桌面應用** — 基於 Wails v2 的原生 macOS/Windows/Linux 視窗，使用 IPC 通訊
- **一步配置** — 桌面版自動配置 SQLite + 管理員帳號，僅需設定模型 API Key
- **應用分享** — 將應用匯出為 ZIP 套件（app.json + meta.json），一鍵匯入執行
- **多模型支援** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 供應商統一介面
- **Agent 工具呼叫** — 可擴展工具註冊表：manage_flows、manage_models、execute_command、read_document、web_search
- **Web 模式** — 透過 `cmd/server/main.go` 啟動為瀏覽器伺服器
- **多語言** — English, 简体中文, 繁體中文, 日本語

## 快速開始

### 桌面應用

```bash
# 前置條件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 一鍵開發模式
wails dev

# 構建 macOS .app
wails build
```

構建產物在 `build/bin/go-ai-agent.app`，雙擊啟動。首次執行自動配置 SQLite 並建立預設管理員帳號（admin/admin），僅需配置模型 API Key。

### Web 伺服器模式

```bash
go build -o go-ai-agent-server ./cmd/server/
./go-ai-agent-server
```

開啟 `http://localhost:19009`，首次執行進入設定精靈。

## 系統架構

```
桌面模式                          Web 模式
┌──────────────────────┐            ┌──────────────────────┐
│  Native WebView      │            │  瀏覽器               │
│  ┌────────────────┐  │            └─────────┬────────────┘
│  │  React 前端     │  │                      │ HTTP/WS
│  │  (嵌入式)       │  │                      │
│  └───────┬────────┘  │            ┌─────────▼────────────┐
└──────────┼───────────┘            │  Go HTTP 伺服器       │
           │ IPC                    │  ├─ REST API         │
┌──────────▼──────────────────────┐ │  ├─ WebSocket        │
│  Go HTTP 伺服器 :19009          │ │  ├─ Agent + 工具      │
│  ├─ REST API + CORS             │ │  └─ 流程引擎          │
│  ├─ IPC 事件 (Wails)            │ └──────────────────────┘
│  ├─ Agent + 工具                │
│  └─ 流程引擎 (DAG)              │
└─────────────────────────────────┘
```

**桌面模式**：使用 Wails IPC 進行通訊（無需 WebSocket）  
**Web 模式**：使用 WebSocket 進行即時通訊

## 專案結構

```
go-ai-agent/
├── main.go                  # 桌面入口 (Wails)
├── cmd/server/main.go       # Web 伺服器入口
├── internal/
│   ├── app/                 # 共享設定（配置、桌面初始化、CORS）
│   ├── agent/               # Agent 迴圈和工具註冊表
│   │   └── tool/            # 工具實作
│   ├── ai/                  # AI 服務
│   │   └── chat/            # 統一聊天服務 + 28+ 供應商
│   ├── entity/              # 資料庫實體（FlowDefinition, AIModel 等）
│   ├── model/               # 資料存取層
│   ├── rest/                # REST 介面
│   ├── runner/              # ChatRunner, FlowRunner
│   └── flow/                # 流程引擎
│       ├── engine/          # DAG 執行器、任務管理器、函數註冊表
│       ├── nodes/           # 節點實作（16 種類型）
│       └── export/          # ZIP 匯入/匯出
├── view/                    # React 前端
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, ModelManager, SetupWizard
│       ├── components/      # 共享元件（ModelForm, IpcAdapter）
│       ├── stores/          # Zustand 狀態管理
│       └── i18n/            # 多語言檔案（en, zh, zh-TW, ja）
├── wails.json               # Wails 專案配置
├── Makefile                 # 建置命令
└── dev.bat                  # 一鍵桌面開發啟動器 (Windows)
```

## 流程引擎

**16 種節點類型**：`start`, `end`, `llm`, `skill`, `user_input`, `condition`, `switch`, `transform`, `split`, `for_each`, `iterator`, `loop`, `script`, `execute`, `image_gen`, `audio_gen`, `video_gen`

**技能節點**：直接執行 prompt，支援模型選擇
```json
{ "prompt": "{{start.output}}", "model": "1.default" }
```

**腳本節點**使用 Starlark（Python 方言）：
```python
# 條件節點：返回 bool → "yes"/"no" 分支
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# 切換節點：返回 string → 路由到匹配的 source_handle
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

**通用批次處理** — ForEach 和 Iterator 呼叫任意註冊的函數：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```
ForEach 並行執行，Iterator 順序執行（跳過失敗項）。

**執行節點**執行本地 shell 命令，可配置逾時（`0` = 無限制）。

**應用匯出**使用 ZIP 格式（`app.json` + `meta.json`）。

## 通訊協定

### 桌面模式 (IPC)
- 使用 Wails Events 進行即時通訊
- 事件模式：`chat:{sessionId}:{type}`（如 `chat:5:chunk`）
- 事件類型：`chunk`, `tool_call`, `tool_result`, `error`, `session_created`

### Web 模式 (WebSocket)
- 連線到 `ws://localhost:19009/ws/chat`
- 訊息類型：
  - `chat` / `agent` — 發送到 ChatRunner
  - `flow_start` / `flow_user_response` / `flow_stop` — 流程執行控制
  - 回應：`chunk`, `tool_call`, `tool_result`, `error`, `session_created`

## 技術棧

| 層級 | 技術 |
|------|------|
| 桌面外殼 | Wails v2 (系統 WebView) |
| 後端 | Go + go-web-frame + CORS 中介軟體 |
| 前端 | React 18 + TypeScript + Vite |
| 流程編輯器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 國際化 | react-i18next |
| 資料庫 | SQLite (桌面版) / MySQL / PostgreSQL (Web 版) |
| 通訊方式 | IPC (桌面版) / WebSocket (Web 版) |

## License

MIT
