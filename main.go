package main

import (
	"net/http"
	"time"

	"github.com/chuccp/go-ai-agent/internal/app"
	"github.com/chuccp/go-web-frame/log"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	web, chatRunner := app.CreateDesktop()
	if web == nil {
		return
	}

	// Start HTTP server in background
	go func() {
		if err := web.Start(); err != nil {
			log.PanicErrors("Desktop service startup failed", err)
		}
	}()

	// Wait for HTTP server to be ready
	waitForReady()

	// Launch native window via Wails
	win := newApp()
	win.chatRunner = chatRunner

	assetOpts := &assetserver.Options{
		Assets: assetFS(),
	}
	if !isWailsDev() {
		assetOpts.Handler = assetHandler()
	}

	err := wails.Run(&options.App{
		Title:        "Go AI Agent",
		Width:        1200,
		Height:       800,
		AssetServer:  assetOpts,
		OnStartup:    win.startup,
		OnShutdown:   win.shutdown,
		Bind:         []interface{}{win},
	})
	if err != nil {
		log.PanicErrors("Wails startup failed", err)
	}
}

func waitForReady() {
	client := &http.Client{Timeout: 2 * time.Second}
	for i := 0; i < 50; i++ {
		resp, err := client.Get("http://localhost:19009/api/setup/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				log.Info("HTTP server ready")
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Warn("HTTP server startup timed out, continuing to launch window")
}
