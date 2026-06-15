# Go AI Agent

> 🚧 **开发中** — 项目正在活跃开发，欢迎使用 Claude Fable 5 来帮助完善功能，非常感谢！

跨平台桌面 AI 智能体平台。**通过聊天创建 AI 工作流** — 用自然语言描述你的需求，AI 智能体为你设计、构建并执行流水线。

基于 **Wails v2** + **React** + **Go** 构建。

[English](README.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

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
- **可视化流程设计器** — Dify 风格拖拽式 DAG 编辑器，16 种节点类型，需要时可手动编辑
- **脚本节点** — 条件和切换节点使用 Starlark（Python 方言）表达式，可访问所有上游数据
- **通用批处理** — ForEach 和 Iterator 节点调用任意函数，不局限于 LLM
- **桌面应用** — 基于 Wails v2 的原生 macOS/Windows/Linux 窗口，使用 IPC 通信
- **一步配置** — 桌面版自动配置 SQLite + 管理员账号，仅需设置模型 API Key
- **应用分享** — 将应用导出为 ZIP 包（app.json + meta.json），一键导入运行
- **多模型支持** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 提供商统一接口
- **Agent 工具调用** — 可扩展工具注册表：manage_flows、manage_models、execute_command、read_document、web_search
- **Web 模式** — 通过 `cmd/server/main.go` 启动为浏览器服务器
- **多语言** — English, 简体中文, 繁體中文, 日本語

## 快速开始

### 桌面应用

```bash
# 前置条件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 一键开发模式
wails dev

# 构建 macOS .app
wails build
```

构建产物在 `build/bin/go-ai-agent.app`，双击启动。首次运行自动配置 SQLite 并创建默认管理员账号（admin/admin），仅需配置模型 API Key。

### Web 服务器模式

```bash
go build -o go-ai-agent-server ./cmd/server/
./go-ai-agent-server
```

访问 `http://localhost:19009`，首次运行进入设置向导。

## 系统架构

```
桌面模式                          Web 模式
┌──────────────────────┐            ┌──────────────────────┐
│  Native WebView      │            │  浏览器               │
│  ┌────────────────┐  │            └─────────┬────────────┘
│  │  React 前端     │  │                      │ HTTP/WS
│  │  (嵌入式)       │  │                      │
│  └───────┬────────┘  │            ┌─────────▼────────────┐
└──────────┼───────────┘            │  Go HTTP 服务器       │
           │ IPC                    │  ├─ REST API         │
┌──────────▼──────────────────────┐ │  ├─ WebSocket        │
│  Go HTTP 服务器 :19009          │ │  ├─ Agent + 工具      │
│  ├─ REST API + CORS             │ │  └─ 流程引擎          │
│  ├─ IPC 事件 (Wails)            │ └──────────────────────┘
│  ├─ Agent + 工具                │
│  └─ 流程引擎 (DAG)              │
└─────────────────────────────────┘
```

**桌面模式**：使用 Wails IPC 进行通信（无需 WebSocket）  
**Web 模式**：使用 WebSocket 进行实时通信

## 项目结构

```
go-ai-agent/
├── main.go                  # 桌面入口 (Wails)
├── cmd/server/main.go       # Web 服务器入口
├── internal/
│   ├── app/                 # 共享设置（配置、桌面初始化、CORS）
│   ├── agent/               # Agent 循环和工具注册表
│   │   └── tool/            # 工具实现
│   ├── ai/                  # AI 服务
│   │   └── chat/            # 统一聊天服务 + 28+ 提供商
│   ├── entity/              # 数据库实体（FlowDefinition, AIModel 等）
│   ├── model/               # 数据访问层
│   ├── rest/                # REST 接口
│   ├── runner/              # ChatRunner, FlowRunner
│   └── flow/                # 流程引擎
│       ├── engine/          # DAG 执行器、任务管理器、函数注册表
│       ├── nodes/           # 节点实现（16 种类型）
│       └── export/          # ZIP 导入/导出
├── view/                    # React 前端
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, ModelManager, SetupWizard
│       ├── components/      # 共享组件（ModelForm, IpcAdapter）
│       ├── stores/          # Zustand 状态管理
│       └── i18n/            # 多语言文件（en, zh, zh-TW, ja）
├── wails.json               # Wails 项目配置
├── Makefile                 # 构建命令
└── dev.bat                  # 一键桌面开发启动器 (Windows)
```

## 流程引擎

**16 种节点类型**：`start`, `end`, `llm`, `skill`, `user_input`, `condition`, `switch`, `transform`, `split`, `for_each`, `iterator`, `loop`, `script`, `execute`, `image_gen`, `audio_gen`, `video_gen`

**技能节点**：直接执行 prompt，支持模型选择
```json
{ "prompt": "{{start.output}}", "model": "1.default" }
```

**脚本节点**使用 Starlark（Python 方言）：
```python
# 条件节点：返回 bool → "yes"/"no" 分支
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# 切换节点：返回 string → 路由到匹配的 source_handle
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

**通用批处理** — ForEach 和 Iterator 调用任意注册的函数：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```
ForEach 并行执行，Iterator 顺序执行（跳过失败项）。

**执行节点**运行本地 shell 命令，可配置超时（`0` = 无限制）。

**应用导出**使用 ZIP 格式（`app.json` + `meta.json`）。

## 通信协议

### 桌面模式 (IPC)
- 使用 Wails Events 进行实时通信
- 事件模式：`chat:{sessionId}:{type}`（如 `chat:5:chunk`）
- 事件类型：`chunk`, `tool_call`, `tool_result`, `error`, `session_created`

### Web 模式 (WebSocket)
- 连接到 `ws://localhost:19009/ws/chat`
- 消息类型：
  - `chat` / `agent` — 发送到 ChatRunner
  - `flow_start` / `flow_user_response` / `flow_stop` — 流程执行控制
  - 响应：`chunk`, `tool_call`, `tool_result`, `error`, `session_created`

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面外壳 | Wails v2 (系统 WebView) |
| 后端 | Go + go-web-frame + CORS 中间件 |
| 前端 | React 18 + TypeScript + Vite |
| 流程编辑器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 国际化 | react-i18next |
| 数据库 | SQLite (桌面版) / MySQL / PostgreSQL (Web 版) |
| 通信方式 | IPC (桌面版) / WebSocket (Web 版) |

## License

MIT
