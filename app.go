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
	"github.com/chuccp/go-web-frame/log"
	wf "github.com/chuccp/go-web-frame"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed view/dist/*
var embeddedAssets embed.FS

// App is the Wails application struct.
type App struct {
	ctx        context.Context
	chatRunner *runner.ChatRunner
	webFrame   *wf.WebFrame
}

func newApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Start HTTP server with the Wails runtime context so that
	// *core.Context embeds it and wailsRuntime.EventsEmit(r.ctx, ...) works.
	go func() {
		if err := a.webFrame.Run(ctx); err != nil {
			log.PanicErrors("Desktop service startup failed", err)
		}
	}()
	waitForReady()
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

// FlowRespond sends a user response to a paused flow execution (desktop IPC bridge).
func (a *App) FlowRespond(executionID uint, response string) string {
	if a.chatRunner == nil {
		return `{"error":"ChatRunner not initialized"}`
	}
	if err := a.chatRunner.HandleFlowUserResponse(executionID, response); err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return `{"ok":true}`
}

// FlowStart begins a flow execution via desktop IPC.
// Flow events are delivered via Wails runtime events on the "flow:<execID>:<type>" channel
// (and the session-scoped "chat:<sessionID>:flow_event" channel).
func (a *App) FlowStart(flowID uint, sessionID uint, initialInput string, formValuesJSON string) string {
	if a.chatRunner == nil {
		return `{"error":"ChatRunner not initialized"}`
	}
	execID, err := a.chatRunner.StartFlowIPC(flowID, sessionID, initialInput, formValuesJSON)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return fmt.Sprintf(`{"execution_id":%d}`, execID)
}

// FlowStop aborts a running flow execution via desktop IPC.
func (a *App) FlowStop(executionID uint) string {
	if a.chatRunner == nil {
		return `{"error":"ChatRunner not initialized"}`
	}
	if err := a.chatRunner.StopFlowIPC(executionID); err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return `{"ok":true}`
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
