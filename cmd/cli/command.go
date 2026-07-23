package main

import (
	agent "github.com/chuccp/go-ai-agent"
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
	var chatConfigs []*ChatConfig
	err := ctx.GetConfig().UnmarshalKey(configKey, chatConfigs)
	if err != nil {
		return err
	}
	receiver.chatManager.GetChat("default")
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
