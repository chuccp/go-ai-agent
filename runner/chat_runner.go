package runner

import (
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
	ctx          *core.Context
	chatService  *chat.UnifiedChatService
	sessionModel *model.ChatSessionModel
	messageModel *model.ChatMessageModel
	flowModel    *model.FlowModel
	nodeModel    *model.FlowNodeModel
	edgeModel    *model.FlowEdgeModel
	flowRunner   *FlowRunner
	activeConns  map[*websocket.Conn]bool
	mu           sync.Mutex
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

	// Load AI model configs from DB and configure chat providers
	aiModel := core.GetModel[*model.AIModelModel](ctx)
	if aiModel != nil {
		models, err := aiModel.List()
		if err == nil {
			for _, m := range models {
				r.chatService.ConfigureProvider(m.Provider, m.APIKey, m.Model, m.BaseURL)
			}
		}
	}

	log.Info("WebSocket 聊天服务已初始化")
	return nil
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
