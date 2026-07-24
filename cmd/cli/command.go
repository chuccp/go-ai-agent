package main

import (
	agent "github.com/chuccp/go-ai-agent"
	"github.com/chuccp/go-ai-agent/internal/api/chat/anthropic"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/util"
)

const configKey = "api.chat"

type Command struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
	chat        *agent.Chat
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
		if util.EqualsAnyIgnoreCase(chatConfig.Type, anthropic.TYPE) {
			receiver.chatManager.RegisterLLM(provider, anthropic.NewService(&anthropic.Config{
				BaseURL: chatConfig.BaseUrl,
				APIKey:  chatConfig.ApiKey,
				Model:   chatConfig.Model,
			}), chatConfig.Model, chatConfig.Default)
		}
	}
	receiver.chat = receiver.chatManager.GetChat("11111")
	return nil
}

func (receiver *Command) HandleMessage(msg string) bool {

	receiver.chat.SendText(msg)

	return false
}
func (receiver *Command) ReadMessage() string {
	return ""
}

func (receiver *Command) Run() error {
	return Run(receiver.ctx, receiver)
}
