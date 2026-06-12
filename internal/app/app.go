package app

import (
	"github.com/chuccp/go-ai-agent/ai/chat"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/rest"
	"github.com/chuccp/go-ai-agent/runner"
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
// webMode=true skips desktop-specific services.
func Create(webMode bool) *wf.WebFrame {
	loadConfig, isFirstRun := loadOrCreateConfig(webMode)

	builder := wf.NewBuilder(loadConfig)

	services := []core.IService{
		&cache.Cache{},
		chat.NewDefaultChatService(),
	}

	builder.Service(services...)

	chatRunner := runner.NewChatRunner()
	builder.Runner(chatRunner)

	flowRun := runner.NewFlowRunner()

	rests := []core.IRest{
		&rest.Api{},
		rest.NewChatRest(chatRunner, flowRun),
	}

	if isFirstRun {
		log.Info("First run detected, enabling setup wizard", zap.String("configPath", configFilePath))
		rests = append(rests, rest.NewSetupRest(configFilePath))
	} else {
		rests = append(rests,
			rest.NewFlowRest(flowRun),
			rest.NewModelRest(),
		)
	}

	builder.Rest(rests...)

	builder.Model(
		&model.ChatSessionModel{},
		&model.ChatMessageModel{},
		&model.AIModelModel{},
		&model.FlowModel{},
		&model.FlowNodeModel{},
		&model.FlowEdgeModel{},
		&model.FlowExecutionModel{},
		&model.AdminUserModel{},
	)
	builder.Filter(cors.NewCrosFilter())
	return builder.Build()
}

// CreateDesktop builds the web framework for desktop mode (with desktop services).
func CreateDesktop() *wf.WebFrame {
	loadConfig, isFirstRun := loadOrCreateConfig(false)

	builder := wf.NewBuilder(loadConfig)

	services := []core.IService{
		&cache.Cache{},
		chat.NewDefaultChatService(),
		&DesktopInitService{},
	}

	builder.Service(services...)

	chatRunner := runner.NewChatRunner()
	builder.Runner(chatRunner)

	flowRun := runner.NewFlowRunner()

	rests := []core.IRest{
		&rest.Api{},
		rest.NewChatRest(chatRunner, flowRun),
	}

	if isFirstRun {
		log.Info("First run detected, enabling setup wizard", zap.String("configPath", configFilePath))
		rests = append(rests, rest.NewSetupRest(configFilePath))
	} else {
		rests = append(rests,
			rest.NewFlowRest(flowRun),
			rest.NewModelRest(),
		)
	}

	builder.Rest(rests...)

	builder.Model(
		&model.ChatSessionModel{},
		&model.ChatMessageModel{},
		&model.AIModelModel{},
		&model.FlowModel{},
		&model.FlowNodeModel{},
		&model.FlowEdgeModel{},
		&model.FlowExecutionModel{},
		&model.AdminUserModel{},
	)
	builder.Filter(cors.NewCrosFilter())
	return builder.Build()
}

func loadOrCreateConfig(webMode bool) (*config.Config, bool) {
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
	cfg.Put("system.apiPrefix", "/api")
	cfg.Put("system.debug", true)
	cfg.Put("system.init", false)

	// Desktop mode: pre-configure SQLite for auto-initialization
	if !webMode {
		cfg.Put("web.db.type", "sqlite")
		cfg.Put("web.db.path", "./data/go-ai-agent.db")
		cfg.Put("system.desktop", true)
	}

	return cfg, true
}
