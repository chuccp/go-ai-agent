package main

import (
	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-agent-sdk/chat"
	"github.com/chuccp/go-ai-agent/internal/api/chat/anthropic"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/util"
)

const configKey = "api.chat"

type Command struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
	chat        *agent.ChatClient
}

func (receiver *Command) Init(ctx *core.Context) error {
	receiver.ctx = ctx
	receiver.chatManager = agent.NewChatManager()
	chatConfigs, err := core.UnmarshalKeyConfig[[]*ChatConfig](configKey, ctx)
	if err != nil {
		return err
	}
	receiver.chatManager.AddTool(agent.NewCommandTool())
	for _, chatConfig := range chatConfigs {
		provider := chatConfig.Name + "_" + chatConfig.Type + "_" + chatConfig.Model
		if util.EqualsAnyIgnoreCase(chatConfig.Type, anthropic.TYPE...) {
			receiver.chatManager.RegisterLLM(provider, anthropic.NewService(&anthropic.Config{
				BaseURL: chatConfig.BaseUrl,
				APIKey:  chatConfig.ApiKey,
				Model:   chatConfig.Model,
			}), chatConfig.Default)
		}
	}
	receiver.chat = receiver.chatManager.GetChat("cli")
	return nil
}

func (receiver *Command) HandleMessage(msg string) bool {
	err := receiver.chat.SendText(msg)
	return err == nil
}
func (receiver *Command) ReadEvent() *chat.ClientEvent {
	return receiver.chat.ReadEvent()
}

func (receiver *Command) Run() error {
	return Run(receiver.ctx, receiver)
}
