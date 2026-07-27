package main

import (
	"time"

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
	chat        *agent.ChatClient
}

func (receiver *Command) Init(ctx *core.Context) error {
	receiver.ctx = ctx
	receiver.chatManager = agent.NewChatManager()
	chatConfigs, err := core.UnmarshalKeyConfig[[]*ChatConfig](configKey, ctx)
	if err != nil {
		return err
	}
	for _, chatConfig := range chatConfigs {
		provider := chatConfig.Name + "_" + chatConfig.Type + "_" + chatConfig.Model
		if util.EqualsAnyIgnoreCase(chatConfig.Type, anthropic.TYPE) {
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

// ReadMessage 阻塞读取事件直到 done/error，返回完整响应文本
func (receiver *Command) ReadMessage() string {
	var result string
	for {
		event := receiver.chat.ReadEvent()
		if event == nil {
			time.Sleep(30 * time.Millisecond)
			continue
		}
		switch event.Type {
		case agent.EventTypeChunk:
			result += event.Content
		case agent.EventTypeError:
			return "[Error] " + event.Message
		case agent.EventTypeDone:
			return result
		}
	}
}

// TryReadEvent 非阻塞读取单个事件，无新事件返回 nil
func (receiver *Command) TryReadEvent() *agent.Event {
	return receiver.chat.ReadEvent()
}

func (receiver *Command) Run() error {
	return Run(receiver.ctx, receiver)
}
