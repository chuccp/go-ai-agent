//go:build !web

package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// App is the Wails application struct.
type App struct {
	ctx context.Context
}

func newApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(_ context.Context) {}

// assetHandler returns a reverse proxy to the Go HTTP server for production use.
func assetHandler() http.Handler {
	target, _ := url.Parse("http://localhost:19009")
	return httputil.NewSingleHostReverseProxy(target)
}
