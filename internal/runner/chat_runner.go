package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/chuccp/go-ai-agent/internal/agent/tool"
	"github.com/chuccp/go-ai-agent/internal/agent/question"
	"github.com/chuccp/go-ai-agent/internal/ai"
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	aiTypes "github.com/chuccp/go-ai-agent/internal/ai/types"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/service"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/gorilla/websocket"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

// connState holds per-connection state for thread-safe WebSocket writes
// and agent cancellation. gorilla/websocket requires all writes to a
// connection to be serialized, so the mutex protects concurrent writes
// from the agent goroutine, flow event broadcaster, and ping ticker.
type connState struct {
	mu     sync.Mutex
	cancel context.CancelFunc // cancels the active agent chat (nil if idle)
}

type ChatRunner struct {
	core.IRunner
	ctx              *core.Context
	chatService      *chat.UnifiedChatService
	genService       *ai.GenService
	sessionModel     *model.ChatSessionModel
	messageModel     *model.ChatMessageModel
	flowModel        *model.FlowModel
	flowService      *service.FlowService
	flowRunner       *FlowRunner
	questionSvc      *question.Service
	activeConns      map[*websocket.Conn]*connState
	defaultModelPath string
	providersLoaded  bool
	isDesktop        bool
	mu               sync.Mutex
}

func NewChatRunner() *ChatRunner {
	return &ChatRunner{activeConns: make(map[*websocket.Conn]*connState)}
}

// QuestionService returns the question service (for desktop IPC bindings).
func (r *ChatRunner) QuestionService() *question.Service {
	return r.questionSvc
}

// maxActiveConns is the maximum number of simultaneous WebSocket connections
// the chat runner will accept. New connections beyond this limit are rejected
// with an error message to prevent resource exhaustion.
const maxActiveConns = 100

func (r *ChatRunner) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatService = core.GetService[*chat.UnifiedChatService](ctx)
	r.genService = core.GetService[*ai.GenService](ctx)
	r.sessionModel = core.GetModel[*model.ChatSessionModel](ctx)
	r.messageModel = core.GetModel[*model.ChatMessageModel](ctx)
	r.flowModel = core.GetModel[*model.FlowModel](ctx)
	r.flowService = core.GetService[*service.FlowService](ctx)

	// Check if running in desktop mode
	r.isDesktop = ctx.GetConfig().GetBoolOrDefault("system.desktop", false)

	// Question service (opencode-style ask/reply). The onAsk callback broadcasts
	// "question_asked" events to the frontend so the UI can render the question.
	// In desktop mode this is emitted via Wails events; in web mode via WS.
	r.questionSvc = question.NewService(func(req question.Request) {
		if r.isDesktop {
			eventName := fmt.Sprintf("chat:%d:question_asked", req.SessionID)
			wailsRuntime.EventsEmit(r.ctx, eventName, req)
		} else {
			data, _ := json.Marshal(map[string]any{
				"type":       "question_asked",
				"session_id": req.SessionID,
				"question":   req,
			})
			r.mu.Lock()
			conns := make([]*websocket.Conn, 0, len(r.activeConns))
			for conn := range r.activeConns {
				conns = append(conns, conn)
			}
			r.mu.Unlock()
			for _, conn := range conns {
				r.sendRaw(conn, data)
			}
		}
	})

	toolRegistry := core.GetService[*tool.Registry](ctx)
	if toolRegistry != nil {
		toolRegistry.SetFlowHandler(r.handleFlowAction)
		toolRegistry.SetFlowExecutionHandler(r.handleFlowExecutionAction)
		toolRegistry.SetModelHandler(r.handleModelAction)
		toolRegistry.SetQuestionService(r.questionSvc)
	}

	r.flowRunner = core.GetRunner[*FlowRunner](ctx)
	if r.flowRunner != nil {
		r.flowRunner.SetSendFunc(func(data []byte) {
			// Parse the event to get execution ID and type
			var event map[string]any
			if err := json.Unmarshal(data, &event); err != nil {
				log.Warn("failed to parse flow event", zap.Error(err))
				return
			}

			executionId, _ := event["execution_id"].(float64)
			eventType, _ := event["type"].(string)

			// Always broadcast to WebSocket connections (works in both web and
			// desktop-dev mode). In pure desktop mode with no WS connections,
			// this is a no-op.
			r.mu.Lock()
			conns := make([]*websocket.Conn, 0, len(r.activeConns))
			for conn := range r.activeConns {
				conns = append(conns, conn)
			}
			r.mu.Unlock()
			if len(conns) > 0 {
				log.Info("flow event broadcasting to WS",
					zap.Int("conns", len(conns)),
					zap.String("type", eventType),
					zap.Uint("execID", uint(executionId)))
				for _, conn := range conns {
					r.sendRaw(conn, data)
				}
			}

			// Additionally emit via Wails events in desktop mode (for the
			// webview IPC adapter that listens on Wails event channels).
			if r.isDesktop {
				eventName := fmt.Sprintf("flow:%d:%s", uint(executionId), eventType)
				wailsRuntime.EventsEmit(r.ctx, eventName, event)
				sessionId, _ := event["session_id"].(float64)
				if sessionId > 0 {
					genericName := fmt.Sprintf("chat:%d:flow_event", uint(sessionId))
					log.Info("flow event emitting via Wails",
						zap.String("event", genericName),
						zap.String("type", eventType),
						zap.Uint("execID", uint(executionId)))
					wailsRuntime.EventsEmit(r.ctx, genericName, event)
				} else {
					log.Warn("flow event has no session_id, cannot route via Wails",
						zap.String("type", eventType),
						zap.Uint("execID", uint(executionId)))
				}
			}
		})
	}
	// If system is already initialized, load providers from DB immediately.
	// Otherwise defer until after setup completes (checked in Run() tick).
	if ctx.GetConfig().GetBoolOrDefault("system.init", false) {
		r.loadProvidersFromDB()
	}

	log.Info("WebSocket chat service initialized")
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
		// Register chat provider
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

		// Register generation providers based on model category.
		// Only image/voice/video models get their respective gen providers;
		// LLM and other categories do not need generation endpoints.
		if r.genService != nil {
			registered := false
			switch m.Category {
			case aiTypes.CategoryImage:
				r.genService.RegisterImageProvider(m.Id, ai.NewOpenAIImageProvider(m.Provider))
				registered = true
			case aiTypes.CategoryVoice:
				r.genService.RegisterSpeechProvider(m.Id, ai.NewOpenAISpeechProvider(m.Provider))
				registered = true
			case aiTypes.CategoryVideo:
				r.genService.RegisterVideoProvider(m.Id, ai.NewOpenAIVideoProvider(m.Provider))
				registered = true
			}
			if registered {
				if err := r.genService.ConfigureProvider(m.Id, m.APIKey, m.BaseURL); err != nil {
					log.Warn("gen provider configure failed",
						zap.Uint("id", m.Id),
						zap.String("provider", m.Provider),
						zap.Error(err))
				}
			}
		}
	}
	// Resolve default model path for fallback when client sends no model
	if def, err := aiModel.FindDefault(aiTypes.CategoryLLM); err == nil && def != nil {
		path := strconv.FormatUint(uint64(def.Id), 10) + "." + def.Model
		r.defaultModelPath = path
		r.chatService.SetDefaultPath(path)
	}
	r.providersLoaded = true
	r.chatService.MarkProvidersReady()
	log.Info("providers loaded from DB", zap.Int("count", len(models)))
}

// ResetProviders clears all loaded providers so the Run() loop will re-fetch
// them from the database on the next tick. Called after a database clear.
func (r *ChatRunner) ResetProviders() {
	r.providersLoaded = false
	r.defaultModelPath = ""
	log.Info("providers reset, will reload from DB on next tick")
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

type Attachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "image/png", "application/pdf", ...
	Size int64  `json:"size"`
	Path string `json:"path"` // Server-side file path
}

type WSRequest struct {
	Type        string               `json:"type"`
	SessionID   uint                 `json:"session_id,omitempty"`
	Model       string               `json:"model,omitempty"`
	Messages    []common.ChatMessage `json:"messages,omitempty"`
	Stream      bool                 `json:"stream"`
	Options     map[string]any       `json:"options,omitempty"`
	Attachments []Attachment         `json:"attachments,omitempty"`
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

// ==================== WebSocket entry point ====================

func (r *ChatRunner) HandleFlowUserResponse(executionId uint, response string) error {
	if r.flowRunner == nil {
		return fmt.Errorf("FlowRunner not initialized")
	}
	return r.flowRunner.HandleUserResponse(executionId, response)
}

// StartFlowIPC starts a flow execution from desktop IPC (no WebSocket connection).
// formValuesJSON is an optional JSON string of form values; pass "" for none.
// The caller is responsible for creating or reusing a chat session.
func (r *ChatRunner) StartFlowIPC(flowID uint, sessionID uint, initialInput string, formValuesJSON string) (uint, error) {
	if r.flowRunner == nil {
		return 0, fmt.Errorf("FlowRunner not initialized")
	}
	opts := FlowStartOptions{InitialInput: initialInput}
	if formValuesJSON != "" {
		var fv map[string]any
		if err := json.Unmarshal([]byte(formValuesJSON), &fv); err == nil {
			opts.FormValues = fv
		}
	}
	return r.flowRunner.HandleFlowStart(flowID, 0, sessionID, opts)
}

// StopFlowIPC aborts a running flow execution from desktop IPC.
func (r *ChatRunner) StopFlowIPC(executionID uint) error {
	if r.flowRunner == nil {
		return fmt.Errorf("FlowRunner not initialized")
	}
	return r.flowRunner.HandleFlowStop(executionID)
}

func (r *ChatRunner) HandleWebSocket(conn *websocket.Conn) error {
	r.mu.Lock()
	if len(r.activeConns) >= maxActiveConns {
		r.mu.Unlock()
		_ = conn.WriteJSON(WSResponse{Type: "error", Message: "Too many concurrent connections. Please try again later."})
		conn.Close()
		log.Warn("websocket connection rejected: max connections reached", zap.Int("limit", maxActiveConns))
		return nil
	}
	state := &connState{}
	r.activeConns[conn] = state
	r.mu.Unlock()

	defer func() {
		// Cancel any active agent chat on this connection
		r.stopAgent(conn)
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
				log.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		var req WSRequest
		if err := json.Unmarshal(message, &req); err != nil {
			r.sendJSON(conn, WSResponse{Type: "error", Message: "Invalid request format: " + err.Error()})
			continue
		}

		switch req.Type {
		case "ping":
			r.sendJSON(conn, WSResponse{Type: "pong"})
		case "chat":
			r.handleChat(conn, req)
		case "agent":
			// Run agent in a goroutine so the WS read loop stays responsive.
			// This allows flow_user_response and stop messages to be processed
			// while the agent (and its blocking tool calls) is still running.
			go r.handleAgent(conn, req)
		case "stop":
			r.stopAgent(conn)
		case "flow_start":
			r.handleFlowStart(conn, req)
		case "flow_user_response":
			r.handleFlowUserResponse(conn, req)
		case "flow_stop":
			r.handleFlowStop(conn, req)
		case "question_reply":
			r.handleQuestionReply(conn, req)
		case "question_reject":
			r.handleQuestionReject(conn, req)
		default:
			r.sendJSON(conn, WSResponse{Type: "error", Message: "Unknown request type: " + req.Type})
		}
	}
	return nil
}

// ── per-conn helpers ──

// handleQuestionReply delivers the user's answers to a blocked ask_user tool call.
func (r *ChatRunner) handleQuestionReply(conn *websocket.Conn, req WSRequest) {
	if r.questionSvc == nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "QuestionService not initialized"})
		return
	}
	var id uint64
	var answers question.Answer
	if req.Options != nil {
		if v, ok := req.Options["question_id"]; ok {
			switch n := v.(type) {
			case float64:
				id = uint64(n)
			case int:
				id = uint64(n)
			}
		}
		if a, ok := req.Options["answers"].([]any); ok {
			answers = make(question.Answer, 0, len(a))
			for _, item := range a {
				if arr, ok := item.([]any); ok {
					labels := make([]string, 0, len(arr))
					for _, l := range arr {
						if s, ok := l.(string); ok {
							labels = append(labels, s)
						}
					}
					answers = append(answers, labels)
				}
			}
		}
	}
	if id == 0 {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "question_id is required"})
		return
	}
	if err := r.questionSvc.Reply(id, answers); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
	}
}

// handleQuestionReject cancels a pending question (user dismissed it).
func (r *ChatRunner) handleQuestionReject(conn *websocket.Conn, req WSRequest) {
	if r.questionSvc == nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "QuestionService not initialized"})
		return
	}
	var id uint64
	if req.Options != nil {
		if v, ok := req.Options["question_id"]; ok {
			switch n := v.(type) {
			case float64:
				id = uint64(n)
			case int:
				id = uint64(n)
			}
		}
	}
	if id == 0 {
		r.sendJSON(conn, WSResponse{Type: "error", Message: "question_id is required"})
		return
	}
	if err := r.questionSvc.Reject(id); err != nil {
		r.sendJSON(conn, WSResponse{Type: "error", Message: err.Error()})
	}
}

// getConnState returns the connState for a connection, or nil if not tracked.
func (r *ChatRunner) getConnState(conn *websocket.Conn) *connState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeConns[conn]
}

// setAgentCancel stores/clears the cancel function for the active agent chat
// on this connection. Called by handleAgent.
func (r *ChatRunner) setAgentCancel(conn *websocket.Conn, cancel context.CancelFunc) {
	state := r.getConnState(conn)
	if state == nil {
		return
	}
	state.mu.Lock()
	state.cancel = cancel
	state.mu.Unlock()
}

// stopAgent cancels the active agent chat on this connection (if any).
func (r *ChatRunner) stopAgent(conn *websocket.Conn) {
	state := r.getConnState(conn)
	if state == nil {
		return
	}
	state.mu.Lock()
	cancel := state.cancel
	state.cancel = nil
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// sendRaw writes raw bytes to a connection under the per-conn write mutex.
func (r *ChatRunner) sendRaw(conn *websocket.Conn, data []byte) {
	state := r.getConnState(conn)
	if state == nil {
		log.Warn("sendRaw: conn not in activeConns, dropping message",
			zap.Int("dataLen", len(data)))
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Warn("websocket write failed", zap.Error(err))
	}
}

// sendJSON marshals and sends a WSResponse to a connection (thread-safe).
func (r *ChatRunner) sendJSON(conn *websocket.Conn, resp WSResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	r.sendRaw(conn, data)
}

func (r *ChatRunner) sendPing() {
	r.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(r.activeConns))
	for conn := range r.activeConns {
		conns = append(conns, conn)
	}
	r.mu.Unlock()
	for _, conn := range conns {
		state := r.getConnState(conn)
		if state == nil {
			continue
		}
		state.mu.Lock()
		conn.WriteMessage(websocket.PingMessage, nil)
		state.mu.Unlock()
	}
}

func (r *ChatRunner) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for conn, state := range r.activeConns {
		state.mu.Lock()
		if state.cancel != nil {
			state.cancel()
		}
		state.mu.Unlock()
		conn.Close()
	}
}
