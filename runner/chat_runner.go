package runner

import (
	"strconv"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/chuccp/go-ai-agent/agent/tool"
	"github.com/chuccp/go-ai-agent/chat"
	"github.com/chuccp/go-ai-agent/chat/common"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type ChatRunner struct {
	core.IRunner
	ctx              *core.Context
	chatService      *chat.UnifiedChatService
	sessionModel     *model.ChatSessionModel
	messageModel     *model.ChatMessageModel
	flowModel        *model.FlowModel
	nodeModel        *model.FlowNodeModel
	edgeModel        *model.FlowEdgeModel
	flowRunner       *FlowRunner
	activeConns      map[*websocket.Conn]bool
	defaultModelPath string
	providersLoaded  bool
	mu               sync.Mutex
}

func NewChatRunner() *ChatRunner {
	return &ChatRunner{activeConns: make(map[*websocket.Conn]bool)}
}

func (r *ChatRunner) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatService = core.GetService[*chat.UnifiedChatService](ctx)
	r.sessionModel = core.GetModel[*model.ChatSessionModel](ctx)
	r.messageModel = core.GetModel[*model.ChatMessageModel](ctx)
	r.flowModel = core.GetModel[*model.FlowModel](ctx)
	r.nodeModel = core.GetModel[*model.FlowNodeModel](ctx)
	r.edgeModel = core.GetModel[*model.FlowEdgeModel](ctx)

	tool.SetFlowHandler(r.handleFlowAction)
	tool.SetModelHandler(r.handleModelAction)

	// If system is already initialized, load providers from DB immediately.
	// Otherwise defer until after setup completes (checked in Run() tick).
	if ctx.GetConfig().GetBoolOrDefault("system.init", false) {
		r.loadProvidersFromDB()
	}

	log.Info("WebSocket 聊天服务已初始化")
	return nil
}

func (r *ChatRunner) loadProvidersFromDB() {
	aiModel := core.GetModel[*model.AIModelModel](r.ctx)
	if aiModel == nil {
		return
	}
	models, err := aiModel.List()
	if err != nil {
		log.Warn("failed to list AI models from DB", zap.Error(err))
		return
	}
	for _, m := range models {
		provider, err := chat.NewProvider(m.Provider)
		if err != nil {
			log.Warn("unknown provider type, skipping",
				zap.String("provider", m.Provider),
				zap.Uint("id", m.Id),
				zap.Error(err))
			continue
		}
		r.chatService.RegisterProvider(m.Id, provider)
		if err := r.chatService.ConfigureProvider(m.Id, m.Provider, m.APIKey, m.Model, m.BaseURL); err != nil {
			log.Warn("provider configure failed",
				zap.Uint("id", m.Id),
				zap.String("provider", m.Provider),
				zap.Error(err))
		}
	}
	// Resolve default model path for fallback when client sends no model
	if def, err := aiModel.FindDefault("llm"); err == nil && def != nil {
		path := strconv.FormatUint(uint64(def.Id), 10) + ".default"
		r.defaultModelPath = path
		r.chatService.SetDefaultPath(path)
	}
	r.providersLoaded = true
	log.Info("providers loaded from DB", zap.Int("count", len(models)))
}

func (r *ChatRunner) SetFlowRunner(fr *FlowRunner) {
	r.flowRunner = fr
	if fr != nil {
		fr.SetSendFunc(func(data []byte) {
			r.mu.Lock()
			defer r.mu.Unlock()
			for conn := range r.activeConns {
				conn.WriteMessage(websocket.TextMessage, data)
				break
			}
		})
	}
}

func (r *ChatRunner) Run() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			r.closeAll()
			return nil
		case <-ticker.C:
			// Lazy-load providers after setup completes (system.init flips to true)
			if !r.providersLoaded && r.ctx.GetConfig().GetBoolOrDefault("system.init", false) {
				r.loadProvidersFromDB()
			}
			r.sendPing()
		}
	}
}

type WSRequest struct {
	Type      string             `json:"type"`
	SessionID uint               `json:"session_id,omitempty"`
	Model     string             `json:"model,omitempty"`
	Messages  []common.ChatMessage `json:"messages,omitempty"`
	Stream    bool               `json:"stream"`
	Options   map[string]any     `json:"options,omitempty"`
}

type WSResponse struct {
	Type           string `json:"type"`
	Content        string `json:"content,omitempty"`
	Reasoning      string `json:"reasoning,omitempty"`
	Done           bool   `json:"done,omitempty"`
	SessionID      uint   `json:"session_id,omitempty"`
	Message        string `json:"message,omitempty"`
	Iteration      int    `json:"iteration"`
	ConversationID string `json:"conversation_id,omitempty"`
}

// ==================== WebSocket 入口 ====================

func (r *ChatRunner) HandleWebSocket(conn *websocket.Conn) error {
	r.mu.Lock()
	r.activeConns[conn] = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		delete(r.activeConns, conn)
		r.mu.Unlock()
		conn.Close()
	}()

	conn.SetReadLimit(10 * 1024 * 1024)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Error("WebSocket 读取错误", zap.Error(err))
			}
			break
		}

		var req WSRequest
		if err := sonic.Unmarshal(message, &req); err != nil {
			r.sendJSON(conn, WSResponse{Type: "error", Message: "无效的请求格式: " + err.Error()})
			continue
		}

		switch req.Type {
		case "ping":
			r.sendJSON(conn, WSResponse{Type: "pong"})
		case "chat":
			r.handleChat(conn, req)
		case "agent":
			r.handleAgent(conn, req)
		case "flow_start":
			r.handleFlowStart(conn, req)
		case "flow_user_response":
			r.handleFlowUserResponse(conn, req)
		case "flow_stop":
			r.handleFlowStop(conn, req)
		default:
			r.sendJSON(conn, WSResponse{Type: "error", Message: "未知的请求类型: " + req.Type})
		}
	}
	return nil
}

func (r *ChatRunner) sendPing() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conn := range r.activeConns {
		conn.WriteMessage(websocket.PingMessage, nil)
	}
}

func (r *ChatRunner) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conn := range r.activeConns {
		conn.Close()
	}
}
