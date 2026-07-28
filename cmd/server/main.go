package main

import (
	"github.com/chuccp/go-ai-agent/cmd/server/model"
	"github.com/chuccp/go-ai-agent/cmd/server/rest"
	"github.com/chuccp/go-ai-agent/cmd/server/runner"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

func main() {
	loadConfig, err := config.LoadConfig("./application.yml")
	if err != nil {
		log.Panic("loadConfig", zap.String("err", err.Error()))
		return
	}
	builder := wf.NewBuilder(loadConfig)
	builder.Runner(&runner.ChatRunner{})
	builder.Rest(&rest.ChatRest{})
	builder.Model(&model.ChatMessageModel{}, &model.ChatSessionModel{})
	frame := builder.Build()
	err = frame.Start()
	if err != nil {
		log.Panic("frame.Start", zap.String("err", err.Error()))
		return
	}
}
