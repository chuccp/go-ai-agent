package main

import (
	"github.com/chuccp/go-ai-agent/ai/chat"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/rest"
	"github.com/chuccp/go-ai-agent/runner"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/component/cache"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

const configFilePath = "application.yml"

func Create() *wf.WebFrame {
	loadConfig, isFirstRun := loadOrCreateConfig()

	builder := wf.NewBuilder(loadConfig)

	builder.Service(
		&cache.Cache{},
		chat.NewDefaultChatService(),
	)

	chatRunner := runner.NewChatRunner()
	builder.Runner(chatRunner)

	flowRun := runner.NewFlowRunner()

	// Build REST handler list.
	// During first-run, only Api, ChatRest (for /ws/chat), and SetupRest are registered.
	// Model-dependent endpoints return errors instead of panicking thanks to nil guards.
	// FlowRest and ModelRest are only registered when fully initialized.
	rests := []core.IRest{
		&rest.Api{},
		rest.NewChatRest(chatRunner, flowRun),
	}

	if isFirstRun {
		log.Info("检测到首次运行，启用初始化向导", zap.String("configPath", configFilePath))
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

	return builder.Build()
}

// loadOrCreateConfig loads application.yml if it exists, or creates an in-memory
// default config for first-run setup. Returns the config and a boolean indicating
// whether the system is in first-run mode (not initialized).
func loadOrCreateConfig() (*config.Config, bool) {
	// Try to load the existing config file
	loadConfig, err := config.LoadConfig(configFilePath)
	if err == nil {
		// File exists — check if fully initialized
		init := loadConfig.GetBoolOrDefault("system.init", false)
		if init {
			return loadConfig, false
		}
		// File exists but init is false — still in setup mode
		log.Info("配置文件存在但系统未完成初始化，进入设置模式")
		return loadConfig, true
	}

	// File doesn't exist — first run, create defaults
	log.Info("未找到 application.yml，进入首次运行初始化模式")
	cfg := config.NewConfig()
	cfg.Put("system.apiPrefix", "/api")
	cfg.Put("system.debug", true)
	cfg.Put("system.init", false)
	return cfg, true
}

func main() {
	web := Create()
	if web == nil {
		return
	}
	err := web.Start()
	if err != nil {
		log.PanicErrors("启动失败", err)
		return
	}
}
