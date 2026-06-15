# Go AI Agent

> 🚧 **開発中** — プロジェクトは活発に開発中です。貢献を歓迎します！

クロスプラットフォームデスクトップ AI エージェントプラットフォーム。**チャットでアプリを作成** — 自然言語でやりたいことを説明するだけで、エージェントがフローを設計・構築・実行します。

**Wails v2** + **React** + **Go** で構築。

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

![Screenshot](screenshot.webp)

## コアコンセプト：アプリ = フロー

Go AI Agent では、**アプリはフローです**。独立した「パッケージ」や「スキル」管理はありません — アプリを作成し、チャットまたは視覚的にそのフローを設計し、実行します。すべてが単一の概念の下に統合されています。

- **アプリ (App)**：ノード、エッジ、設定を含む完全で自己完結したワークフロー
- **フロー (Flow)**：アプリのロジックの視覚的表現
- **スキルノード (Skill Node)**：フロー内で直接プロンプトを実行するノード（外部スキル管理不要）

## 機能

- **チャットでアプリ作成** — 自然言語会話を通じて完全なワークフローを構築
- **視覚的フローデザイナー** — 17種類のノードタイプをサポートするドラッグ＆ドロップDAGエディタ
- **17種類のノードタイプ**：start, end, llm, skill, user_input, condition, switch, transform, split, for_each, iterator, loop, script, execute, image_gen, audio_gen, video_gen
- **スキルノード** — フロー内で直接プロンプトを実行（外部スキル管理不要）
- **スクリプトベースのロジック** — 条件とスイッチノードは Starlark（Python方言）を使用して複雑な分岐を実装
- **バッチ処理** — 配列処理用の for_each（並列）と iterator（逐次）ノード
- **デスクトップアプリ** — Wails v2 によるネイティブ Windows/macOS/Linux、IPC通信を使用
- **Webモード** — ブラウザベースのサーバーとして実行、WebSocket通信を使用
- **ワンステップ設定** — デスクトップモードは SQLite + 管理者アカウントを自動設定
- **アプリエクスポート** — アプリをZIPパッケージとしてエクスポート、ワンクリックでインポート
- **マルチモデル** — OpenAI、Claude、Gemini、DeepSeekなど28以上のプロバイダーに統一インターフェースで対応
- **Agentツール実行** — 拡張可能なツールレジストリ：manage_flows、manage_models、execute_command、read_document、web_search
- **多言語** — English, 简体中文, 繁體中文, 日本語

## クイックスタート

### デスクトップアプリ

```bash
# 前提条件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# 開発モード（ホットリロード）
make desktop-dev  # macOS/Linux
dev.bat           # Windows

# 本番アプリをビルド
make desktop-build  # macOS/Linux
wails build         # 手動
```

初回実行時に SQLite を自動設定し、デフォルトの管理者アカウント（admin/admin）を作成します。モデル API キーを設定するだけです。

### Webサーバーモード

```bash
make server-build  # macOS/Linux
go build -o go-ai-agent-server.exe ./cmd/server/  # Windows

./go-ai-agent-server
```

`http://localhost:19009` を開きます — 初回実行時にセットアップウィザードが表示されます。

## アーキテクチャ

### デスクトップモード (IPC)
```
┌─────────────────────────────────┐
│  Native WebView (Wails v2)      │
│  ┌───────────────────────────┐  │
│  │  React フロントエンド      │  │
│  │  - ChatHome               │  │
│  │  - FlowDesigner           │  │
│  │  - ModelManager           │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ Wails IPC (Events)
┌─────────────┼───────────────────┐
│  Go バックエンド :19009         │
│  ├─ REST API (/api/*)           │
│  ├─ IPC イベントバス            │
│  ├─ Agent + ツール              │
│  └─ フローエンジン (DAG実行)    │
└─────────────────────────────────┘
```

### Webモード (WebSocket)
```
┌─────────────────────────────────┐
│  ブラウザ                        │
│  ┌───────────────────────────┐  │
│  │  React フロントエンド      │  │
│  └──────────┬────────────────┘  │
└─────────────┼───────────────────┘
              │ WebSocket (/ws/chat)
              │ HTTP (/api/*)
┌─────────────┼───────────────────┐
│  Go バックエンド :19009         │
│  ├─ REST API                    │
│  ├─ WebSocket サーバー          │
│  ├─ Agent + ツール              │
│  └─ フローエンジン              │
└─────────────────────────────────┘
```

**通信プロトコル：**
- デスクトップ：Wails IPC イベント（例：`chat:{sessionId}:chunk`）
- Web：WebSocket メッセージ（JSON形式）

## ノードタイプ

### 基本ノード
- **start**：フローの開始点
- **end**：フローの終了点
- **user_input**：ユーザー入力または確認を待機

### AIノード
- **llm**：プロンプトとシステムメッセージでLLMを呼び出し
- **skill**：プロンプトを直接実行（簡略化されたLLMノード）
- **image_gen**：AIモデルで画像を生成
- **audio_gen**：AIモデルで音声/スピーチを生成
- **video_gen**：AIモデルで動画を生成

### ロジックノード
- **condition**：if/else分岐（Starlarkブール式）
- **switch**：多方向分岐（Starlark文字列式）
- **loop**：条件を満たすまで繰り返し実行

### データ処理ノード
- **transform**：Goテンプレートベースのデータ変換
- **split**：区切り文字でテキストをJSON配列に分割
- **for_each**：配列項目を並列処理
- **iterator**：配列項目を逐次処理

### 実行ノード
- **script**：Starlark Pythonカスタムコード
- **execute**：ローカルシェルコマンドを実行

## スクリプトベースノード

**条件**と**スイッチ**ノードは Starlark（Python方言）を使用：

```python
# 条件：boolを返す → "yes"/"no"分岐
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# スイッチ：stringを返す → 一致するsource_handleにルーティング
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

## バッチ処理

**for_each** は並列実行：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

**iterator** は逐次実行（失敗をスキップ）：
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```

## スキルノード

スキルノードはプロンプトを直接実行 — 外部スキル管理不要：

```json
{
  "prompt": "以下のテキストを要約：\n\n{{llm.output}}",
  "model": "1.default"
}
```

スキルノードは本質的に簡略化されたLLMノードで、フロー内で素早くプロンプトを実行するために使用されます。

## アプリエクスポート形式

アプリはZIPパッケージとしてエクスポートされ、以下を含みます：
- `meta.json`：アプリメタデータ（名前、アイコン、説明）
- `app.json`：ノードとエッジを含むフロー定義
- `resources/`：追加ファイル（あれば）

```bash
# エクスポート：アプリ → ZIPファイル
# インポート：ZIPファイル → アプリ
```

## プロジェクト構造

```
go-ai-agent/
├── main.go                  # デスクトップエントリー (Wails)
├── cmd/server/main.go       # Webサーバーエントリー
├── internal/
│   ├── agent/               # エージェントループとツールレジストリ
│   │   └── tool/            # ツール実装
│   ├── ai/                  # AIサービス
│   │   └── chat/            # 統合チャットサービス + 28以上のプロバイダー
│   ├── app/                 # アプリケーション設定と構成
│   ├── config/              # 設定管理
│   ├── entity/              # データベースエンティティ（FlowDefinition, AIModelなど）
│   ├── flow/                # フローエンジン
│   │   ├── engine/          # DAG実行、タスクマネージャー、関数レジストリ
│   │   ├── nodes/           # 17種類のノードタイプ実装
│   │   └── export/          # アプリのエクスポート/インポート（ZIP形式）
│   ├── model/               # データアクセス層
│   ├── rest/                # REST APIエンドポイント
│   ├── runner/              # ChatRunner, FlowRunner
│   ├── service/             # ビジネスロジックサービス
│   └── util/                # ユーティリティ
├── view/                    # Reactフロントエンド
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, FlowRunner, ModelManager, SetupWizard
│       ├── components/      # 共有コンポーネント（ModelForm, IpcAdapterなど）
│       ├── stores/          # Zustand状態ストア
│       └── i18n/            # ロケールファイル（en, zh, zh-TW, ja）
├── wails.json               # Wailsプロジェクト設定
├── Makefile                 # ビルドターゲット
└── dev.bat                  # ワンクリックデスクトップ開発ランチャー（Windows）
```

## 技術スタック

| 階層 | 技術 |
|------|------|
| デスクトップシェル | Wails v2（システムWebView） |
| バックエンド | Go + go-web-frame + CORSミドルウェア |
| フロントエンド | React 18 + TypeScript + Vite |
| フローエディタ | reactflow + Zustand |
| チャットUI | @assistant-ui/react |
| 国際化 | react-i18next |
| データベース | SQLite（デスクトップ）/ MySQL / PostgreSQL（Web） |
| 通信 | IPC（デスクトップ）/ WebSocket（Web） |

## ライセンス

MIT
