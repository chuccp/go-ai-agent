# Go AI Agent

> 🚧 **開發中** — 專案正在活躍開發，歡迎貢獻！

跨平台桌面 AI 智慧體平台。**透過聊天建立應用** — 用自然語言描述你的需求，AI 智慧體為你設計、建構並執行流程。

基於 **Wails v2** + **React** + **Go** 建構。

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

## 核心概念：應用 = 流程

在 Go AI Agent 中，**應用就是流程**。沒有獨立的「套件」或「技能」管理 — 你建立一個應用，透過聊天或視覺化方式設計其流程，然後執行它。所有概念統一起在單一概念下。

- **應用 (App)**：一個完整的、自包含的工作流程，包含節點、邊和配置
- **流程 (Flow)**：應用邏輯的視覺化表示
- **技能節點 (Skill Node)**：直接在流程中執行 prompt 的節點（無需外部技能管理）

## 功能特色

- **聊天建立應用** — 透過自然語言對話建構完整的工作流程
- **視覺化流程設計器** — 拖拽式 DAG 編輯器，支援 17 種節點類型
- **17 種節點類型**：start, end, llm, skill, user_input, condition, switch, transform, split, for_each, iterator, loop, script, execute, image_gen, audio_gen, video_gen
- **技能節點** — 直接在流程中執行 prompt（無需外部技能管理）
- **基於腳本的邏輯** — 條件和切換節點使用 Starlark（Python 方言）實現複雜分支
- **批次處理** — for_each（平行）和 iterator（順序）節點用於陣列處理
- **桌面應用** — 透過 Wails v2 實現原生 Windows/macOS/Linux，使用 IPC 通訊
- **Web 模式** — 作為基於瀏覽器的伺服器執行，使用 WebSocket 通訊
- **一步配置** — 桌面模式自動配置 SQLite + 管理員帳號
- **應用匯出** — 將應用匯出為 ZIP 套件，一鍵匯入
- **多模型支援** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 供應商統一介面
- **Agent 工具呼叫** — 可擴展工具註冊表：manage_flows、manage_models、execute_command、read_document、web_search
- **多語言** — English, 简体中文, 繁體中文, 日本語

## 快速開始

### 桌面應用

```bash
# 前置條件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 開發模式（熱重載）
make desktop-dev  # macOS/Linux
dev.bat           # Windows

# 建構生產應用
make desktop-build  # macOS/Linux
wails build         # 手動
```

首次執行自動配置 SQLite 並建立預設管理員帳號（admin/admin）。你只需要配置模型 API 金鑰。

### Web 伺服器模式

```bash
make server-build  # macOS/Linux
go build -o go-ai-agent-server.exe ./cmd/server/  # Windows

./go-ai-agent-server
```

開啟 `http://localhost:19009` — 首次執行會開啟設定精靈。

## 系統架構

### 桌面模式 (IPC)
```
┌─────────────────────────────────┐
│  Native WebView (Wails v2)      │
│  ┌───────────────────────────┐  │
│  │  React 前端               │  │
│  │  - ChatHome               │  │
│  │  - FlowDesigner           │  │
│  │  - ModelManager           │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ Wails IPC (Events)
┌─────────────┼───────────────────┐
│  Go 後端 :19009                 │
│  ├─ REST API (/api/*)           │
│  ├─ IPC 事件總線                │
│  ├─ Agent + 工具                │
│  └─ 流程引擎 (DAG 執行器)       │
└─────────────────────────────────┘
```

### Web 模式 (WebSocket)
```
┌─────────────────────────────────┐
│  瀏覽器                         │
│  ┌───────────────────────────┐  │
│  │  React 前端               │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ WebSocket (/ws/chat)
              │ HTTP (/api/*)
┌─────────────┼───────────────────┐
│  Go 後端 :19009                 │
│  ├─ REST API                    │
│  ├─ WebSocket 伺服器            │
│  ├─ Agent + 工具                │
│  └─ 流程引擎                    │
└─────────────────────────────────┘
```

**通訊協定：**
- 桌面：Wails IPC 事件（例如 `chat:{sessionId}:chunk`）
- Web：WebSocket 訊息（JSON 格式）

## 節點類型

### 基礎節點
- **start**：流程入口點
- **end**：流程出口點
- **user_input**：等待使用者輸入或確認

### AI 節點
- **llm**：使用 prompt 和系統訊息呼叫 LLM
- **skill**：直接執行 prompt（簡化的 LLM 節點）
- **image_gen**：透過 AI 模型產生圖像
- **audio_gen**：透過 AI 模型產生音訊/語音
- **video_gen**：透過 AI 模型產生影片

### 邏輯節點
- **condition**：if/else 分支（Starlark 布林表達式）
- **switch**：多路分支（Starlark 字串表達式）
- **loop**：重複執行直到滿足條件

### 資料處理節點
- **transform**：基於 Go 模板的資料轉換
- **split**：按分隔符號將文字拆分為 JSON 陣列
- **for_each**：平行處理陣列項
- **iterator**：順序處理陣列項

### 執行節點
- **script**：Starlark Python 自訂程式碼
- **execute**：執行本機 shell 命令

## 基於腳本的節點

**條件**和**切換**節點使用 Starlark（Python 方言）：

```python
# 條件：返回 bool → "yes"/"no" 分支
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# 切換：返回 string → 路由到匹配的 source_handle
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

## 批次處理

**for_each** 平行執行：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

**iterator** 順序執行（跳過失敗）：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

## 技能節點

技能節點直接執行 prompt — 無需外部技能管理：

```json
{
  "prompt": "總結以下文字：\n\n{{llm.output}}",
  "model": "1.default"
}
```

技能節點本質上是簡化的 LLM 節點，用於在流程中快速執行 prompt。

## 應用匯出格式

應用匯出為 ZIP 套件，包含：
- `meta.json`：應用元資料（名稱、圖示、描述）
- `app.json`：帶有節點和邊的流程定義
- `resources/`：附加檔案（如果有）

```bash
# 匯出：應用 → ZIP 檔案
# 匯入：ZIP 檔案 → 應用
```

## 專案結構

```
go-ai-agent/
├── main.go                  # 桌面入口 (Wails)
├── cmd/server/main.go       # Web 伺服器入口
├── internal/
│   ├── agent/               # Agent 迴圈和工具註冊表
│   │   └── tool/            # 工具實現
│   ├── ai/                  # AI 服務
│   │   └── chat/            # 統一聊天服務 + 28+ 供應商
│   ├── app/                 # 應用設定和配置
│   ├── config/              # 配置管理
│   ├── entity/              # 資料庫實體（FlowDefinition, AIModel 等）
│   ├── flow/                # 流程引擎
│   │   ├── engine/          # DAG 執行器、任務管理器、函數註冊表
│   │   ├── nodes/           # 17 種節點類型實現
│   │   └── export/          # 應用匯出/匯入（ZIP 格式）
│   ├── model/               # 資料存取層
│   ├── rest/                # REST API 端點
│   ├── runner/              # ChatRunner, FlowRunner
│   ├── service/             # 業務邏輯服務
│   └── util/                # 工具類
├── view/                    # React 前端
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, FlowRunner, ModelManager, SetupWizard
│       ├── components/      # 共享元件（ModelForm, IpcAdapter 等）
│       ├── stores/          # Zustand 狀態儲存
│       └── i18n/            # 語言檔案（en, zh, zh-TW, ja）
├── wails.json               # Wails 專案配置
├── Makefile                 # 建構目標
└── dev.bat                  # 一鍵桌面開發啟動器（Windows）
```

## 技術棧

| 層級 | 技術 |
|------|------|
| 桌面外殼 | Wails v2 (系統 WebView) |
| 後端 | Go + go-web-frame + CORS 中介軟體 |
| 前端 | React 18 + TypeScript + Vite |
| 流程編輯器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 國際化 | react-i18next |
| 資料庫 | SQLite (桌面) / MySQL / PostgreSQL (Web) |
| 通訊 | IPC (桌面) / WebSocket (Web) |

## 授權條款

MIT
