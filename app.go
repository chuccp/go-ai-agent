package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
)

//go:embed view/dist/*
var embeddedAssets embed.FS

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

// assetFS returns the embedded frontend assets or nil in dev mode.
func assetFS() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "view/dist")
	if err != nil {
		return nil
	}
	return sub
}

// assetHandler returns a reverse proxy to the Go HTTP server for production use.
func assetHandler() http.Handler {
	target, _ := url.Parse("http://localhost:19009")
	return httputil.NewSingleHostReverseProxy(target)
}

