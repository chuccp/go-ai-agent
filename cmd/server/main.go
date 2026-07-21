package main

import (
	"flag"

	"github.com/chuccp/go-ai-agent/internal2/app"
	"github.com/chuccp/go-web-frame/log"
)

func main() {
	flag.Parse()
	web := app.Create()
	if web == nil {
		return
	}
	if err := web.Start(); err != nil {
		log.PanicErrors("Startup failed", err)
	}
}
