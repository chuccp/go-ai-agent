package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-ai-agent/cmd/server/runner"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
	"go.uber.org/zap"
)

const maxActiveConns = 100

// connState tracks per-WebSocket-connection state: the active chat client
// and a cancel function to stop the event relay goroutine.
type connState struct {
	client *agent.ChatClient
	cancel context.CancelFunc
}

// ChatRest registers WebSocket and REST API routes for the web chat.
// It handles all WebSocket I/O (connect, read, write, disconnect) and
// delegates to the go-agent-sdk ChatRunner for the actual LLM work.
type ChatRest struct {
	context     *core.Context
	chatRunner  *runner.ChatRunner
	activeConns map[string]*connState
	mu          sync.Mutex
	connSeq     int64
}

// NewChatRest creates a new ChatRest instance.
func NewChatRest() *ChatRest {
	return &ChatRest{
		activeConns: make(map[string]*connState),
	}
}

// Init registers all chat-related routes on the web framework context.
func (c *ChatRest) Init(ctx *core.Context) error {
	c.context = ctx
	c.chatRunner = core.GetRunner[*runner.ChatRunner](ctx)

	// WebSocket — real-time chat streaming
	ctx.WebSocket("/ws/chat", c.HandleWebSocket)

	// REST — health check
	ctx.Get("/api/chat/health", c.health)

	log.Info("Chat REST routes registered (go-agent-sdk)", zap.String("ws", "/ws/chat"))
	return nil
}

// ── WebSocket handler ──────────────────────────────────────────────────

// HandleWebSocket is the entry point for web WebSocket connections.
// Each connection gets a unique session ID; all chat messages within
// one connection share the same conversation context.
func (c *ChatRest) HandleWebSocket(webSocket *web.WebSocket) error {
	stream, err := webSocket.OpenStream(web.WithOriginPatterns("*"))
	if err != nil {
		return err
	}

	connID := c.nextConnID()

	c.mu.Lock()
	if len(c.activeConns) >= maxActiveConns {
		c.mu.Unlock()
		resp, _ := json.Marshal(agent.Event{
			Type:    agent.EventTypeError,
			Message: "Too many concurrent connections. Please try again later.",
		})
		_ = stream.WriteText(context.Background(), resp)
		stream.Close()
		log.Warn("websocket connection rejected: max connections reached",
			zap.Int("limit", maxActiveConns))
		return nil
	}
	state := &connState{}
	c.activeConns[connID] = state
	c.mu.Unlock()

	defer func() {
		c.stopConn(connID)
		c.mu.Lock()
		delete(c.activeConns, connID)
		c.mu.Unlock()
		stream.Close()
	}()

	stream.Conn().SetReadLimit(10 * 1024 * 1024)

	for {
		_, message, err := stream.Read(stream.Context())
		if err != nil {
			log.Debug("WebSocket read ended", zap.String("connID", connID), zap.Error(err))
			break
		}

		var req struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			c.sendJSON(stream, agent.Event{
				Type:    agent.EventTypeError,
				Message: "Invalid JSON format: " + err.Error(),
			})
			continue
		}

		switch req.Type {
		case "ping":
			c.sendJSON(stream, agent.Event{Type: "pong"})
		case "chat":
			if req.Message == "" {
				c.sendJSON(stream, agent.Event{
					Type:    agent.EventTypeError,
					Message: "message field is required for chat",
				})
				continue
			}
			c.handleChat(connID, stream, req.Message)
		case "stop":
			c.stopConn(connID)
		default:
			c.sendJSON(stream, agent.Event{
				Type:    agent.EventTypeError,
				Message: "Unknown request type: " + req.Type,
			})
		}
	}
	return nil
}

// ── Chat helpers ───────────────────────────────────────────────────────

// handleChat sends a user message to the chat session for this connection.
// It stops any previous chat on the same connection first (one active chat at a time).
func (c *ChatRest) handleChat(connID string, stream *web.WebSocketStream, message string) {
	// Stop any existing chat on this connection before starting a new one.
	c.stopConn(connID)

	client := c.chatRunner.GetChat(connID)

	ctx, cancel := context.WithCancel(context.Background())

	c.mu.Lock()
	if state, ok := c.activeConns[connID]; ok {
		state.client = client
		state.cancel = cancel
	}
	c.mu.Unlock()

	if err := client.SendText(message); err != nil {
		c.sendJSON(stream, agent.Event{
			Type:    agent.EventTypeError,
			Message: err.Error(),
		})
		cancel()
		c.mu.Lock()
		if state, ok := c.activeConns[connID]; ok {
			state.client = nil
			state.cancel = nil
		}
		c.mu.Unlock()
		return
	}

	go c.relayEvents(ctx, cancel, client, stream, connID)
}

// relayEvents reads events from the ChatClient in a loop and writes them
// to the WebSocket as JSON. It exits when the chat is done, an error occurs,
// or the context is cancelled (via stop or disconnect).
func (c *ChatRest) relayEvents(ctx context.Context, cancel context.CancelFunc, client *agent.ChatClient, stream *web.WebSocketStream, connID string) {
	defer func() {
		cancel()
		c.mu.Lock()
		if state, ok := c.activeConns[connID]; ok && state.client == client {
			state.client = nil
			state.cancel = nil
		}
		c.mu.Unlock()
	}()

	for {
		event := client.ReadEvent()
		if event == nil {
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		c.sendJSON(stream, *event)

		if event.Type == agent.EventTypeDone || event.Type == agent.EventTypeError {
			return
		}
	}
}

// stopConn cancels the relay goroutine and closes the chat client
// for the given connection, if one is active.
func (c *ChatRest) stopConn(connID string) {
	c.mu.Lock()
	state, ok := c.activeConns[connID]
	c.mu.Unlock()
	if !ok {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	if state.client != nil {
		state.client.Close()
	}
}

// ── Helpers ────────────────────────────────────────────────────────────

// sendJSON marshals an agent.Event and writes it to the WebSocket stream.
func (c *ChatRest) sendJSON(stream *web.WebSocketStream, event agent.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Warn("failed to marshal event", zap.Error(err))
		return
	}
	if err := stream.WriteText(context.Background(), data); err != nil {
		log.Warn("websocket write failed", zap.Error(err))
	}
}

// nextConnID generates a unique connection ID for each WebSocket session.
func (c *ChatRest) nextConnID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connSeq++
	return fmt.Sprintf("web-%d", c.connSeq)
}

// ── REST handlers ──────────────────────────────────────────────────────

// health returns a simple status check for the chat service.
func (c *ChatRest) health(_ *web.Request) (any, error) {
	c.mu.Lock()
	conns := len(c.activeConns)
	c.mu.Unlock()
	return web.Data(map[string]any{
		"status":        "ok",
		"engine":        "go-agent-sdk",
		"active_conns":  conns,
		"server_time":   time.Now().Format(time.RFC3339),
	}), nil
}
