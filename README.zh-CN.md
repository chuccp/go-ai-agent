# Go AI Agent

> 🚧 **开发中** — 项目正在活跃开发，欢迎贡献！

跨平台桌面 AI 智能体平台。**通过聊天创建应用** — 用自然语言描述你的需求，AI 智能体为你设计、构建并执行流程。

基于 **Wails v2** + **React** + **Go** 构建。

[English](README.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

![Screenshot](screenshot.webp)

## 核心概念：应用 = 流程

在 Go AI Agent 中，**应用就是流程**。没有独立的"包"或"技能"管理 — 你创建一个应用，通过聊天或可视化方式设计其流程，然后运行它。所有概念统一在单一概念下。

- **应用 (App)**：一个完整的、自包含的工作流，包含节点、边和配置
- **流程 (Flow)**：应用逻辑的可视化表示
- **技能节点 (Skill Node)**：直接在流程中执行 prompt 的节点（无需外部技能管理）

## 功能特性

- **聊天创建应用** — 通过自然语言对话构建完整的工作流
- **可视化流程设计器** — 拖拽式 DAG 编辑器，支持 17 种节点类型
- **17 种节点类型**：start, end, llm, skill, user_input, condition, switch, transform, split, for_each, iterator, loop, script, execute, image_gen, audio_gen, video_gen
- **技能节点** — 直接在流程中执行 prompt（无需外部技能管理）
- **基于脚本的逻辑** — 条件和切换节点使用 Starlark（Python 方言）实现复杂分支
- **批处理** — for_each（并行）和 iterator（顺序）节点用于数组处理
- **桌面应用** — 通过 Wails v2 实现原生 Windows/macOS/Linux，使用 IPC 通信
- **Web 模式** — 作为基于浏览器的服务器运行，使用 WebSocket 通信
- **一步配置** — 桌面模式自动配置 SQLite + 管理员账号
- **应用导出** — 将应用导出为 ZIP 包，一键导入
- **多模型支持** — OpenAI、Claude、Gemini、DeepSeek 等 28+ 提供商统一接口
- **Agent 工具调用** — 可扩展工具注册表：manage_flows、manage_models、execute_command、read_document、web_search
- **多语言** — English, 简体中文, 繁體中文, 日本語

## 本项目的优势

1. **聊天即开发** — 不需要写代码，用自然语言描述需求，AI 智能体自动设计、构建并运行工作流。从想法到应用，只需一次对话。

2. **开箱即用** — 下载即用，无需配置数据库、无需部署服务器。桌面模式首次运行自动完成所有初始化，只需填入 API 密钥即可开始使用。

3. **本地运行，数据在手** — 应用和数据全部存储在本地设备，不依赖云端服务。对话记录、流程配置、模型密钥都在你自己的电脑上。

4. **灵活的流程编排** — 可视化拖拽设计复杂工作流，支持条件分支、循环、批处理、并行执行。17 种节点类型满足从简单问答到复杂自动化的各种场景。

5. **多模型自由切换** — 支持 OpenAI、Claude、Gemini、DeepSeek 等 28+ 模型提供商，每个节点可以独立选择模型，灵活组合不同 AI 的能力。

6. **多模态 AI 能力** — 不仅是文本对话，还支持图像生成、音频生成、视频生成，将多种 AI 能力串联在一个流程中。

7. **一键分享与复用** — 应用可导出为 ZIP 包，一键导入到其他实例。社区可以共享工作流模板，开箱即用。

8. **跨平台桌面体验** — Windows、macOS、Linux 原生桌面应用，启动快、体积小、资源占用低，同时支持 Web 浏览器访问。

## 与其他产品有什么不同？

### 与 ChatGPT、Claude 等 AI 对话工具

ChatGPT、Claude 是非常强大的通用 AI 工具，几乎什么都能做。

但纯对话有一个天然问题：**对话越长，上下文越臃肿，模型的注意力越分散**。当你试图在一个长对话中完成多个步骤时，前面的内容会干扰后面的判断，输出质量随对话长度下降。

Go AI Agent 通过**流程节点**解决了这个问题：

- 每个节点只接收上游节点的输出作为上下文，而不是整个对话历史
- 每次 LLM 调用都是**聚焦的、干净的**，不受无关信息干扰
- 不同节点可以选择不同模型，各取所长

这意味着同样的任务，流程化执行的每个步骤都比在长对话中执行**更专注、更准确**。

此外，创建好的流程可以导出分享，别人导入即可运行，不需要重复搭建。

## 快速开始

### 桌面应用

```bash
# 前置条件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 开发模式（热重载）
make desktop-dev  # macOS/Linux
dev.bat           # Windows

# 构建生产应用
make desktop-build  # macOS/Linux
wails build         # 手动
```

首次运行自动配置 SQLite 并创建默认管理员账号（admin/admin）。你只需要配置模型 API 密钥。

### Web 服务器模式

```bash
make server-build  # macOS/Linux
go build -o go-ai-agent-server.exe ./cmd/server/  # Windows

./go-ai-agent-server
```

打开 `http://localhost:19009` — 首次运行会打开设置向导。

## 系统架构

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
│  Go 后端 :19009                 │
│  ├─ REST API (/api/*)           │
│  ├─ IPC 事件总线                │
│  ├─ Agent + 工具                │
│  └─ 流程引擎 (DAG 执行器)       │
└─────────────────────────────────┘
```

### Web 模式 (WebSocket)
```
┌─────────────────────────────────┐
│  浏览器                         │
│  ┌───────────────────────────┐  │
│  │  React 前端               │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ WebSocket (/ws/chat)
              │ HTTP (/api/*)
┌─────────────┼───────────────────┐
│  Go 后端 :19009                 │
│  ├─ REST API                    │
│  ├─ WebSocket 服务器            │
│  ├─ Agent + 工具                │
│  └─ 流程引擎                    │
└─────────────────────────────────┘
```

**通信协议：**
- 桌面：Wails IPC 事件（例如 `chat:{sessionId}:chunk`）
- Web：WebSocket 消息（JSON 格式）

## 节点类型

### 基础节点
- **start**：流程入口点
- **end**：流程出口点
- **user_input**：等待用户输入或确认

### AI 节点
- **llm**：使用 prompt 和系统消息调用 LLM
- **skill**：直接执行 prompt（简化的 LLM 节点）
- **image_gen**：通过 AI 模型生成图像
- **audio_gen**：通过 AI 模型生成音频/语音
- **video_gen**：通过 AI 模型生成视频

### 逻辑节点
- **condition**：if/else 分支（Starlark 布尔表达式）
- **switch**：多路分支（Starlark 字符串表达式）
- **loop**：重复执行直到满足条件

### 数据处理节点
- **transform**：基于 Go 模板的数据转换
- **split**：按分隔符将文本拆分为 JSON 数组
- **for_each**：并行处理数组项
- **iterator**：顺序处理数组项

### 执行节点
- **script**：Starlark Python 自定义代码
- **execute**：运行本地 shell 命令

## 基于脚本的节点

**条件**和**切换**节点使用 Starlark（Python 方言）：

```python
# 条件：返回 bool → "yes"/"no" 分支
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# 切换：返回 string → 路由到匹配的 source_handle
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

## 批处理

**for_each** 并行运行：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

**iterator** 顺序运行（跳过失败）：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

## 技能节点

技能节点直接执行 prompt — 无需外部技能管理：

```json
{
  "prompt": "总结以下文本：\n\n{{llm.output}}",
  "model": "1.default"
}
```

技能节点本质上是简化的 LLM 节点，用于在流程中快速执行 prompt。

## 应用导出格式

应用导出为 ZIP 包，包含：
- `meta.json`：应用元数据（名称、图标、描述）
- `app.json`：带有节点和边的流程定义
- `resources/`：附加文件（如果有）

```bash
# 导出：应用 → ZIP 文件
# 导入：ZIP 文件 → 应用
```

## 项目结构

```
go-ai-agent/
├── main.go                  # 桌面入口 (Wails)
├── cmd/server/main.go       # Web 服务器入口
├── internal/
│   ├── agent/               # Agent 循环和工具注册表
│   │   └── tool/            # 工具实现
│   ├── ai/                  # AI 服务
│   │   └── chat/            # 统一聊天服务 + 28+ 提供商
│   ├── app/                 # 应用设置和配置
│   ├── config/              # 配置管理
│   ├── entity/              # 数据库实体（FlowDefinition, AIModel 等）
│   ├── flow/                # 流程引擎
│   │   ├── engine/          # DAG 执行器、任务管理器、函数注册表
│   │   ├── nodes/           # 17 种节点类型实现
│   │   └── export/          # 应用导出/导入（ZIP 格式）
│   ├── model/               # 数据访问层
│   ├── rest/                # REST API 端点
│   ├── runner/              # ChatRunner, FlowRunner
│   ├── service/             # 业务逻辑服务
│   └── util/                # 工具类
├── view/                    # React 前端
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, FlowRunner, ModelManager, SetupWizard
│       ├── components/      # 共享组件（ModelForm, IpcAdapter 等）
│       ├── stores/          # Zustand 状态存储
│       └── i18n/            # 语言文件（en, zh, zh-TW, ja）
├── wails.json               # Wails 项目配置
├── Makefile                 # 构建目标
└── dev.bat                  # 一键桌面开发启动器（Windows）
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 桌面外壳 | Wails v2 (系统 WebView) |
| 后端 | Go + go-web-frame + CORS 中间件 |
| 前端 | React 18 + TypeScript + Vite |
| 流程编辑器 | reactflow + Zustand |
| 聊天 UI | @assistant-ui/react |
| 国际化 | react-i18next |
| 数据库 | SQLite (桌面) / MySQL / PostgreSQL (Web) |
| 通信 | IPC (桌面) / WebSocket (Web) |

## 许可证

MIT
