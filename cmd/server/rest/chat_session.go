package rest

import (
	"context"
	"sync"

	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-ai-agent/cmd/server/entity"
	"github.com/chuccp/go-ai-agent/cmd/server/runner"
	"github.com/chuccp/go-ai-agent/cmd/server/service"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/util"
	"github.com/chuccp/go-web-frame/web"
	"github.com/coder/websocket"
	"github.com/google/uuid"
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
	context            *core.Context
	chatRunner         *runner.ChatRunner
	chatSessionService *service.ChatSessionService
	activeConns        map[string]*connState
	mu                 sync.Mutex
	connSeq            int64
}

// Init registers all chat-related routes on the web framework context.
func (c *ChatRest) Init(ctx *core.Context) error {
	c.context = ctx
	c.activeConns = make(map[string]*connState)
	c.chatRunner = core.GetRunner[*runner.ChatRunner](ctx)
	c.chatSessionService = core.GetService[*service.ChatSessionService](ctx)

	// Session CRUD
	ctx.Get("/api/chat/sessions", c.listSessions)
	ctx.Post("/api/chat/sessions", c.createSession)
	ctx.Delete("/api/chat/sessions/:id", c.deleteSession)
	ctx.Get("/api/chat/sessions/:id/messages", c.getSessionMessages)
	ctx.WebSocket("/ws/chat", c.HandleWebSocket)
	log.Info("Chat REST routes registered (go-agent-sdk)", zap.String("ws", "/ws/chat"))
	return nil
}

// ── Session REST handlers ─────────────────────────────────────────────

// listSessions returns all chat sessions ordered by most recently updated.
func (c *ChatRest) listSessions(request *web.Request) (any, error) {
	sessions, err := c.chatSessionService.ListSessions()
	if err != nil {
		return nil, err
	}
	return web.Data(sessions), nil
}

// createSession creates a new chat session with an optional title.
func (c *ChatRest) createSession(request *web.Request) (any, error) {
	title := "New Chat"
	if jsonObj, err := request.Json(); err == nil {
		if t := jsonObj.GetString("title"); t != "" {
			title = t
		}
	}

	session, err := c.chatSessionService.CreateSession(request.Ctx(), title)
	if err != nil {
		return nil, err
	}
	return web.Data(session), nil
}

// deleteSession deletes a session and all its messages.
func (c *ChatRest) deleteSession(request *web.Request) (any, error) {
	id := request.ParamUint("id")
	if err := c.chatSessionService.DeleteSession(request.Ctx(), id); err != nil {
		return nil, err
	}
	return web.Ok("deleted"), nil
}

// getSessionMessages returns all messages for a session ordered by creation time.
func (c *ChatRest) getSessionMessages(request *web.Request) (any, error) {
	sessionId := request.ParamUint("id")
	messages, err := c.chatSessionService.GetSessionMessages(sessionId)
	if err != nil {
		return nil, err
	}
	return web.Data(messages), nil
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
	defer stream.Close()
	stream.Conn().SetReadLimit(10 * 1024 * 1024)
	chatId := uuid.New().String()
	for {
		messageType, message, err := stream.Read(stream.Context())
		if err != nil {
			log.Debug("WebSocket read ended", zap.Error(err))
			break
		}
		switch messageType {
		case websocket.MessageText:
			revMessage, err := util.JsonUnmarshal[*entity.RevMessage](message)
			if err != nil {
				return err
			}
			switch revMessage.Type {
			case entity.ChatType:
				c.chatRunner.HandleChat(chatId, revMessage)
			case entity.StopType:
				c.chatRunner.HandleStop(chatId, revMessage)
			}
		case websocket.MessageBinary:

		}

		log.Debug("WebSocket read", zap.String("type", string(messageType)), zap.Any("message", message))
	}
	return nil
}
