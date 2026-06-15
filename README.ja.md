# Go AI Agent

> 🚧 **開発中** — このプロジェクトは活発に開発中です。Claude Fable 5 を使って機能の改善・完成にご協力いただける方を歓迎します。ありがとうございます！

クロスプラットフォームデスクトップ AI エージェントプラットフォーム。**チャットで AI ワークフローを作成** — 自然言語でやりたいことを説明するだけで、エージェントがパイプラインを設計・構築・実行します。

**Wails v2** + **React** + **Go** で構築。

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md)

![Screenshot](screenshot.webp)

## チャットによるワークフロー作成の利点

従来のワークフローツールでは、ビジュアルエディタの使い方を学ぶ必要がありました — ノードをドラッグし、エッジを配線し、パラメータを設定します。Go AI Agent では、必要なことを伝えるだけです：

> *「最新のAIニュースを取得し、DeepSeekで要約し、要約を日本語に翻訳して、保存前に私にレビューさせてください」*

エージェントは**意図を理解 → ノード構造を提案 → 確認を待つ → 自動的にフローを作成**します。手動配線も設定の推測も不要です。会話を通じて反復的に改善できます — 「モデルをGPT-5に変更して」「要約の前に感情分析ステップを追加して」 — エージェントがすぐにフローを更新します。

**手動エディタと比較した利点：**

- **学習不要** — ノードタイプ、接続ルール、設定スキーマを学ぶ必要なし
- **自然な反復** — 同僚と話すように、会話を通じてフローを洗練
- **設計ガイダンス** — エージェントがベストプラクティスを提案（例：「このフローには送信前のユーザー確認ステップが必要です」）
- **高速プロトタイピング** — アイデアから動作するパイプラインまで1分未満
- **完全な制御** — ビジュアルエディタでいつでも手動微調整が可能

## 機能

- **チャットによるワークフロー作成** — 自然言語での会話を通じて `manage_flows` ツールで AI パイプラインを構築
- **ビジュアルフローデザイナー** — Difyスタイルのドラッグ＆ドロップDAGエディタ、16種類のノード、必要に応じて手動編集可能
- **スクリプトノード** — 条件とスイッチノードは Starlark（Python方言）式を使用、全アップストリームデータにアクセス可能
- **汎用バッチ処理** — ForEach と Iterator ノードは任意の関数を呼び出し、LLM に限定されない
- **デスクトップアプリ** — Wails v2によるネイティブ macOS/Windows/Linux ウィンドウ、IPC通信を使用
- **ワンステップ設定** — デスクトップ版はSQLite + 管理者アカウントを自動設定、モデルAPIキーのみ必要
- **アプリ共有** — アプリをZIPパッケージとしてエクスポート（app.json + meta.json）、ワンクリックでインポート実行
- **マルチモデル** — OpenAI、Claude、Gemini、DeepSeekなど28以上のプロバイダーに統一インターフェースで対応
- **Agentツール実行** — 拡張可能なツールレジストリ：manage_flows、manage_models、execute_command、read_document、web_search
- **Webモード** — `cmd/server/main.go` でブラウザサーバーとして起動
- **多言語** — English, 简体中文, 繁體中文, 日本語

## クイックスタート

### デスクトップアプリ

```bash
# 前提条件: Go 1.25+, Node 18+, pnpm
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone https://github.com/chuccp/go-ai-agent.git
cd go-ai-agent

# ワンクリック開発モード
wails dev

# macOS .app をビルド
wails build
```

ビルド成果物は `build/bin/go-ai-agent.app`。ダブルクリックで起動、初回起動時はSQLiteとデフォルト管理者アカウント（admin/admin）を自動設定、モデルAPIキーの設定のみ必要です。

### Webサーバーモード

```bash
go build -o go-ai-agent-server ./cmd/server/
./go-ai-agent-server
```

`http://localhost:19009` を開くと、初回実行時にセットアップウィザードが表示されます。

## アーキテクチャ

```
デスクトップモード                    Webモード
┌──────────────────────┐            ┌──────────────────────┐
│  Native WebView      │            │  ブラウザ             │
│  ┌────────────────┐  │            └─────────┬────────────┘
│  │  React フロント  │  │                      │ HTTP/WS
│  │  (埋め込み)      │  │                      │
│  └───────┬────────┘  │            ┌─────────▼────────────┐
└──────────┼───────────┘            │  Go HTTP サーバー     │
           │ IPC                    │  ├─ REST API         │
┌──────────▼──────────────────────┐ │  ├─ WebSocket        │
│  Go HTTP サーバー :19009        │ │  ├─ Agent + ツール    │
│  ├─ REST API + CORS             │ │  └─ フローエンジン    │
│  ├─ IPC イベント (Wails)        │ └──────────────────────┘
│  ├─ Agent + ツール              │
│  └─ フローエンジン (DAG)        │
└─────────────────────────────────┘
```

**デスクトップモード**: Wails IPC を使用した通信（WebSocket不要）  
**Webモード**: リアルタイム通信に WebSocket を使用

## プロジェクト構成

```
go-ai-agent/
├── main.go                  # デスクトップエントリー (Wails)
├── cmd/server/main.go       # Webサーバーエントリー
├── internal/
│   ├── app/                 # 共有セットアップ（設定、デスクトップ初期化、CORS）
│   ├── agent/               # エージェントループとツールレジストリ
│   │   └── tool/            # ツール実装
│   ├── ai/                  # AIサービス
│   │   └── chat/            # 統合チャットサービス + 28以上のプロバイダー
│   ├── entity/              # データベースエンティティ（FlowDefinition, AIModelなど）
│   ├── model/               # データアクセス層
│   ├── rest/                # RESTエンドポイント
│   ├── runner/              # ChatRunner, FlowRunner
│   └── flow/                # フローエンジン
│       ├── engine/          # DAGエグゼキューター、タスクマネージャー、関数レジストリ
│       ├── nodes/           # ノード実装（16種類）
│       └── export/          # ZIPインポート/エクスポート
├── view/                    # Reactフロントエンド
│   └── src/
│       ├── pages/           # ChatHome, FlowDesigner, ModelManager, SetupWizard
│       ├── components/      # 共有コンポーネント（ModelForm, IpcAdapter）
│       ├── stores/          # Zustand状態管理
│       └── i18n/            # 多言語ファイル（en, zh, zh-TW, ja）
├── wails.json               # Wailsプロジェクト設定
├── Makefile                 # ビルドターゲット
└── dev.bat                  # ワンクリックデスクトップ開発ランチャー (Windows)
```

## フローエンジン

**16種類のノード**: `start`, `end`, `llm`, `skill`, `user_input`, `condition`, `switch`, `transform`, `split`, `for_each`, `iterator`, `loop`, `script`, `execute`, `image_gen`, `audio_gen`, `video_gen`

**スキルノード**: モデル選択でプロンプトを直接実行
```json
{ "prompt": "{{start.output}}", "model": "1.default" }
```

**スクリプトベースノード**は Starlark（Python方言）を使用:
```python
# 条件ノード: bool を返す → "yes"/"no" 分岐
v = ctx["user_input"]["output"].lower()
result = v in ("yes", "confirm", "ok")

# スイッチノード: string を返す → 一致する source_handle にルーティング
score = int(ctx["score"]["output"])
if score >= 90:  result = "A"
elif score >= 60: result = "B"
else:            result = "C"
```

**汎用バッチ処理** — ForEach と Iterator は登録された任意の関数を呼び出す:
```json
{ "items_key": "split", "function": "llm", "args": { "model": "...", "prompt": "{{item.output}}" } }
```
ForEach は並列実行、Iterator は順次実行（失敗をスキップ）。

**実行ノード**はローカルシェルコマンドを実行、設定可能なタイムアウト（`0` = 制限なし）。

**アプリエクスポート**はZIP形式（`app.json` + `meta.json`）を使用。

## 通信プロトコル

### デスクトップモード (IPC)
- Wails Events を使用したリアルタイム通信
- イベントパターン: `chat:{sessionId}:{type}`（例: `chat:5:chunk`）
- イベントタイプ: `chunk`, `tool_call`, `tool_result`, `error`, `session_created`

### Webモード (WebSocket)
- `ws://localhost:19009/ws/chat` に接続
- メッセージタイプ:
  - `chat` / `agent` — ChatRunner に送信
  - `flow_start` / `flow_user_response` / `flow_stop` — フロー実行制御
  - レスポンス: `chunk`, `tool_call`, `tool_result`, `error`, `session_created`

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| デスクトップシェル | Wails v2 (システム WebView) |
| バックエンド | Go + go-web-frame + CORS ミドルウェア |
| フロントエンド | React 18 + TypeScript + Vite |
| フローエディタ | reactflow + Zustand |
| チャット UI | @assistant-ui/react |
| 国際化 | react-i18next |
| データベース | SQLite (デスクトップ) / MySQL / PostgreSQL (Web) |
| 通信方式 | IPC (デスクトップ) / WebSocket (Web) |

## License

MIT
