package app

import (
	"github.com/chuccp/go-ai-agent/internal2/agent/tool"
	"github.com/chuccp/go-ai-agent/internal2/ai"
	"github.com/chuccp/go-ai-agent/internal2/ai/chat"
	"github.com/chuccp/go-ai-agent/internal2/model"
	"github.com/chuccp/go-ai-agent/internal2/rest"
	"github.com/chuccp/go-ai-agent/internal2/runner"
	"github.com/chuccp/go-ai-agent/internal2/service"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/component/cache"
	"github.com/chuccp/go-web-frame/component/cors"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

const configFilePath = "application.yml"

// Create builds the web framework with all services and REST endpoints.
func Create() *wf.WebFrame {
	loadConfig, isFirstRun := loadOrCreateConfig()

	builder := wf.NewBuilder(loadConfig)

	services := []core.IService{
		&cache.Cache{},
		chat.NewDefaultChatService(),
		ai.NewGenService(),
		tool.NewRegistry(),
		&service.FlowService{},
	}

	builder.Service(services...)

	chatRunner := runner.NewChatRunner()
	builder.Runner(chatRunner)
	builder.Runner(runner.NewFlowRunner())

	rests := []core.IRest{
		&rest.Api{},
		rest.NewChatRest(),
		rest.NewSetupRest(configFilePath),
		rest.NewFlowRest(),
		rest.NewModelRest(),
		&rest.SystemRest{},
	}

	if isFirstRun {
		log.Info("First run detected, enabling setup wizard", zap.String("configPath", configFilePath))
	}

	builder.Rest(rests...)

	builder.Model(
		&model.ChatSessionModel{},
		&model.ChatMessageModel{},
		&model.AIModelModel{},
		&model.FlowModel{},
		&model.AdminUserModel{},
	)
	builder.Filter(cors.NewCrosFilter())
	return builder.Build()
}

func loadOrCreateConfig() (*config.Config, bool) {
	loadConfig, err := config.LoadConfig(configFilePath)
	if err == nil {
		init := loadConfig.GetBoolOrDefault("system.init", false)
		if init {
			return loadConfig, false
		}
		log.Info("Config file exists but system not initialized, entering setup mode")
		return loadConfig, true
	}
	log.Info("No application.yml found, entering first-run initialization mode")
	cfg := config.NewConfig()
	cfg.Put("system.debug", true)
	cfg.Put("system.init", false)

	return cfg, true
}
