package main

import (
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

func main() {

	loadConfig, err := config.LoadConfig("application.yml")
	if err != nil {
		log.Error("", zap.Error(err))
		return
	}
	builder := wf.NewBuilder(loadConfig)
	builder.Runner(&Command{})
	frame := builder.Build()
	err0 := frame.Start()
	if err0 != nil {
		return
	}
}
