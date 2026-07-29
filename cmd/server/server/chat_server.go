package server

import (
	"sync"

	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-ai-agent/cmd/server/entity"
	"github.com/chuccp/go-ai-agent/internal/api/chat/anthropic"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/util"
	"go.uber.org/zap"
)

type AgentServer struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
	lock        sync.RWMutex
}

func (r *AgentServer) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatManager = agent.NewChatManager()
	chatConfigs, err := core.UnmarshalKeyConfig[[]*ChatConfig](configKey, ctx)
	if err != nil {
		return err
	}
	r.chatManager.AddTool(agent.NewExecuteCommandTool())
	for _, chatConfig := range chatConfigs {
		provider := chatConfig.Name + "_" + chatConfig.Type + "_" + chatConfig.Model
		if util.EqualsAnyIgnoreCase(chatConfig.Type, anthropic.TYPE...) {
			r.chatManager.RegisterLLM(provider, anthropic.NewService(&anthropic.Config{
				BaseURL: chatConfig.BaseUrl,
				APIKey:  chatConfig.ApiKey,
				Model:   chatConfig.Model,
			}), chatConfig.Default)
		}
	}
	log.Info("AgentServer initialized (go-agent-sdk)", zap.Int("providers", len(chatConfigs)))
	return nil
}
func (r *AgentServer) GetChat() *Chat {
	return newChat(r.chatManager)
}

func (r *AgentServer) HandleChat(chat *agent.ChatClient, message *entity.RevMessage) {

}
func (r *AgentServer) HandleStop(chat *agent.ChatClient, message *entity.RevMessage) {

}
