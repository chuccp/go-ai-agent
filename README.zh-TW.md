# Go AI Agent

跨平台桌面 AI 智能體平台。**透過聊天建立 AI 工作流程** — 用自然語言描述你的需求，AI 智能體為你設計、構建並執行流水線。

基於 **Wails v2** + **React** + **Go** 構建。

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md)

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
- **視覺化流程設計器** — Dify 風格拖曳式 DAG 編輯器，14 種節點類型，需要時可手動編輯
- **桌面應用** — 基於 Wails v2 的原生 macOS/Windows/Linux 視窗，雙擊即可使用
- **一步配置** — 桌面版自動配置 SQLite + 管理員帳號，僅需設定模型 API Key
- **多模型支援** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 供應商統一介面
- **Agent 工具調用** — 內建工具執行迴圈，可擴展工具註冊表
- **串流聊天** — 基於 WebSocket 的即時串流輸出，支援思考過程展示
- **Web 模式** — `--web` 參數啟動為瀏覽器伺服器（SQLite/MySQL/PostgreSQL）
- **多語言** — English, 简体中文, 繁體中文, 日本語

## 快速開始

### 桌面版

```bash
# 前置條件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 開發模式（熱重載）
wails dev

# 構建 macOS .app
wails build
```

構建產物在 `build/bin/go-ai-agent.app`，雙擊啟動。首次執行僅需設定模型 API Key。

### Web 伺服器模式

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
cd view && pnpm dev
```

開啟 `http://localhost:5173`，首次執行進入設定精靈。

## 工作原理

```
你: "建立一個內容審核流程"
         │
         ▼
┌─────────────────────────────────┐
│  AI 智能體 (manage_flows 工具)   │
│                                 │
│  1. 理解 — 詢問關於流程的       │
│     細節問題                    │
│  2. 設計 — 用自然語言提出       │
│     節點結構方案                │
│  3. 確認 — 等待你的明確批准     │
│  4. 建立 — 自動構建包含正確     │
│     節點和連線的流程            │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  自動建立的可視化流程            │
│  ┌───┐   ┌──────┐   ┌───┐     │
│  │開始│──▶│ LLM  │──▶│結束│     │
│  └───┘   └──────┘   └───┘     │
│                                 │
│  隨時可在拖曳式編輯器中手動修改  │
└─────────────────────────────────┘
```

## 技術棧

| 層級 | 技術 |
|------|------|
| 桌面外殼 | Wails v2 (系統 WebView) |
| 後端 | Go + go-web-frame + Gorilla WebSocket |
| 前端 | React 18 + TypeScript + Vite |
| 流程編輯器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 國際化 | react-i18next |
| 資料庫 | SQLite (桌面版) / MySQL / PostgreSQL (Web 版) |

## 專案結構

```
go-ai-agent/
├── main.go              # 入口，--web 參數，桌面/Web 路由
├── app.go               # Wails App 結構體，資源嵌入
├── desktop_init.go      # 桌面版自動初始化 SQLite + 管理員
├── wails.json           # Wails 專案配置
├── Makefile             # 構建命令
├── agent/               # Agent 迴圈，工具註冊表
├── ai/chat/             # 統一聊天服務 + 28+ 供應商
├── runner/              # ChatRunner, FlowRunner
├── rest/                # REST 介面
├── flow/                # 流程引擎 (DAG 執行器, 14 種節點)
└── view/                # React 前端
    └── src/
        ├── pages/       # 聊天首頁, 流程設計器, 模型管理, 設定精靈
        ├── components/  # 聊天 UI, 流程編輯器, 通用元件
        ├── stores/      # Zustand 狀態管理
        └── i18n/        # 多語言檔案
```

## License

MIT
