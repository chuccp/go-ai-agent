//go:build wails

package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/chuccp/go-ai-agent/internal/runner"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed view/dist/*
var embeddedAssets embed.FS

// App is the Wails application struct.
type App struct {
	ctx        context.Context
	chatRunner *runner.ChatRunner
}

func newApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {}

// ── IPC methods (Wails Bind) ──

// AgentChat starts an agent conversation and returns the session ID.
// Streaming results are emitted via Wails runtime events.
func (a *App) AgentChat(sessionID uint, modelPath string, message string, thinkLevel string, flowID uint) string {
	if a.chatRunner == nil {
		return `{"error":"ChatRunner not initialized"}`
	}

	newSessionID, err := a.chatRunner.StartAgentIPC(a.ctx, sessionID, modelPath, message, thinkLevel, flowID)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}

	// Emit session_created if it's a new session
	if sessionID == 0 {
		wailsRuntime.EventsEmit(a.ctx, fmt.Sprintf("chat:%d:session_created", newSessionID), map[string]any{
			"session_id": newSessionID,
		})
	}

	return fmt.Sprintf(`{"session_id":%d}`, newSessionID)
}

// assetFS returns the embedded frontend assets.
func assetFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "view/dist")
	if err != nil {
		return nil
	}
	return sub
}

// assetHandler returns a reverse proxy to the Go HTTP server.
func assetHandler() http.Handler {
	target, _ := url.Parse("http://localhost:19009")
	return httputil.NewSingleHostReverseProxy(target)
}
