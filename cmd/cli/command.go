package main

import (
	agent "github.com/chuccp/go-ai-agent"
	"github.com/chuccp/go-ai-agent/internal/api/chat/anthropic"
	"github.com/chuccp/go-web-frame/core"
)

const configKey = "api.chat"

type Command struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
}

func (receiver *Command) Init(ctx *core.Context) error {
	receiver.ctx = ctx
	receiver.chatManager = agent.NewChatManager()
	chatConfigs, err := core.UnmarshalKeyConfig[[]*ChatConfig](configKey, ctx)
	if err != nil {
		return err
	}
	for _, chatConfig := range chatConfigs {
		provider := chatConfig.Name + "_" + chatConfig.Type
		receiver.chatManager.RegisterLLM(provider, anthropic.NewService(&anthropic.Config{}))

	}
	return nil
}

func (receiver *Command) HandleMessage(msg string) bool {

	return false
}
func (receiver *Command) ReadMessage() string {
	return ""
}

func (receiver *Command) Run() error {
	return Run(receiver.ctx, receiver)
}
