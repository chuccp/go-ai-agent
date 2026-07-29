package runner

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

type ChatRunner struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
	chatMap     *sync.Map
}

func (r *ChatRunner) Init(ctx *core.Context) error {
	r.ctx = ctx
	r.chatMap = new(sync.Map)
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
	log.Info("ChatRunner initialized (go-agent-sdk)", zap.Int("providers", len(chatConfigs)))
	return nil
}

func (r *ChatRunner) Run() error {
	return nil
}
func (r *ChatRunner) GetChat(id string) *agent.ChatClient {
	return r.chatManager.GetChat(id)
}

func (r *ChatRunner) HandleChat(chatId string, message *entity.RevMessage) {

}

func (r *ChatRunner) HandleStop(chatId string, message *entity.RevMessage) {

}
