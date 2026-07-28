package runner

import (
	"time"

	"github.com/chuccp/go-agent-sdk/agent"
	"github.com/chuccp/go-ai-agent/internal/api/chat/anthropic"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/util"
	"go.uber.org/zap"
)

const configKey = "api.chat"

// ChatConfig mirrors the CLI's config structure for LLM provider registration.
type ChatConfig struct {
	Name    string
	Type    string
	BaseUrl string
	ApiKey  string
	Model   string
	Default bool
}

// ChatRunner is a pure chat engine built on go-agent-sdk. It holds the
// ChatManager (LLM providers + tools) and exposes GetChat for callers
// (e.g. REST layer) to create per-session chat clients.
//
// This mirrors cmd/cli/command.go's Init logic — config-based provider
// registration — without any WebSocket or connection concerns.
type ChatRunner struct {
	core.IRunner
	ctx         *core.Context
	chatManager *agent.ChatManager
}

// NewChatRunner creates a new ChatRunner instance.
func NewChatRunner() *ChatRunner {
	return &ChatRunner{}
}

// Init creates the ChatManager, reads LLM provider configs from [api.chat],
// registers the execute_command tool, and registers Anthropic providers.
func (r *ChatRunner) Init(ctx *core.Context) error {
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

	log.Info("ChatRunner initialized (go-agent-sdk)",
		zap.Int("providers", len(chatConfigs)))
	return nil
}

// Run is a no-op lifecycle hook. The ChatRunner has no background work
// of its own — all chat activity is driven through GetChat clients.
func (r *ChatRunner) Run() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return nil
		case <-ticker.C:
			// idle heartbeat — ChatManager sessions live independently
		}
	}
}

// GetChat returns a ChatClient for the given session ID. Sessions are
// created automatically on first access and reused for subsequent calls
// with the same ID, preserving conversation history.
//
// Callers are responsible for the client lifecycle: call SendText to
// submit messages and ReadEvent in a loop to consume streaming events.
func (r *ChatRunner) GetChat(id string) *agent.ChatClient {
	return r.chatManager.GetChat(id)
}
