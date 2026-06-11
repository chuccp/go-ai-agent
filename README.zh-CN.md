# Go AI Agent

跨平台桌面 AI 智能体平台。**通过聊天创建 AI 工作流** — 用自然语言描述你的需求，AI 智能体为你设计、构建并执行流水线。

基于 **Wails v2** + **React** + **Go** 构建。

[English](README.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

## 为什么用聊天创建工作流？

传统的流程工具需要学习可视化编辑器 — 拖拽节点、连接边线、配置参数。使用 Go AI Agent，你只需要告诉它你想要什么：

> *"帮我创建一个流程：抓取最新的 AI 新闻，用 DeepSeek 进行摘要，把摘要翻译成日语，让我审核后再保存。"*

智能体会**理解你的意图 → 提出节点结构方案 → 等待你确认 → 自动创建流程**。不需要手动连线，不需要猜测配置。你可以通过对话逐步优化 — "把模型换成 GPT-5"、"在摘要之前加一步情感分析" — 智能体立即更新流程。

**相比手动编辑器的优势：**

- **零学习成本** — 不需要学习节点类型、连线规则、配置参数
- **自然迭代** — 通过对话优化流程，就像和同事讨论一样
- **设计指导** — 智能体会建议最佳实践（如"这个流程在发送前应加一个用户确认步骤"）
- **快速原型** — 从想法到可运行的流水线，不到一分钟
- **完全可控** — 随时可以在可视化编辑器中手动微调

## 功能特性

- **聊天创建工作流** — 通过自然语言对话，使用 `manage_flows` 工具构建 AI 流水线
- **可视化流程设计器** — Dify 风格拖拽式 DAG 编辑器，14 种节点类型，需要时可手动编辑
- **桌面应用** — 基于 Wails v2 的原生 macOS/Windows/Linux 窗口，双击即用
- **一步配置** — 桌面版自动配置 SQLite + 管理员账号，仅需设置模型 API Key
- **多模型支持** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 提供商统一接口
- **Agent 工具调用** — 内置工具执行循环，可扩展工具注册表
- **流式聊天** — 基于 WebSocket 的实时流式输出，支持思考过程展示
- **Web 模式** — `--web` 参数启动为浏览器服务器（SQLite/MySQL/PostgreSQL）
- **多语言** — English, 简体中文, 繁體中文, 日本語

## 快速开始

### 桌面版

```bash
# 前置条件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 开发模式（热重载）
wails dev

# 构建 macOS .app
wails build
```

构建产物在 `build/bin/go-ai-agent.app`，双击启动。首次运行仅需配置模型 API Key。

### Web 服务器模式

```bash
go build -o go-ai-agent . && ./go-ai-agent --web
cd view && pnpm dev
```

访问 `http://localhost:5173`，首次运行进入设置向导。

## 工作原理

```
你: "创建一个内容审核流程"
         │
         ▼
┌─────────────────────────────────┐
│  AI 智能体 (manage_flows 工具)   │
│                                 │
│  1. 理解 — 询问关于流程的       │
│     细节问题                    │
│  2. 设计 — 用自然语言提出       │
│     节点结构方案                │
│  3. 确认 — 等待你的明确批准     │
│  4. 创建 — 自动构建包含正确     │
│     节点和连线的流程            │
└─────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  自动创建的可视化流程            │
│  ┌───┐   ┌──────┐   ┌───┐     │
│  │开始│──▶│ LLM  │──▶│结束│     │
│  └───┘   └──────┘   └───┘     │
│                                 │
│  随时可在拖拽式编辑器中手动修改  │
└─────────────────────────────────┘
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面外壳 | Wails v2 (系统 WebView) |
| 后端 | Go + go-web-frame + Gorilla WebSocket |
| 前端 | React 18 + TypeScript + Vite |
| 流程编辑器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 国际化 | react-i18next |
| 数据库 | SQLite (桌面版) / MySQL / PostgreSQL (Web 版) |

## 项目结构

```
go-ai-agent/
├── main.go              # 入口，--web 参数，桌面/Web 路由
├── app.go               # Wails App 结构体，资源嵌入
├── desktop_init.go      # 桌面版自动初始化 SQLite + 管理员
├── wails.json           # Wails 项目配置
├── Makefile             # 构建命令
├── agent/               # Agent 循环，工具注册表
├── ai/chat/             # 统一聊天服务 + 28+ 提供商
├── runner/              # ChatRunner, FlowRunner
├── rest/                # REST 接口
├── flow/                # 流程引擎 (DAG 执行器, 14 种节点)
└── view/                # React 前端
    └── src/
        ├── pages/       # 聊天首页, 流程设计器, 模型管理, 设置向导
        ├── components/  # 聊天 UI, 流程编辑器, 通用组件
        ├── stores/      # Zustand 状态管理
        └── i18n/        # 多语言文件
```

## License

MIT
