package app

import (
	"os"
	"path/filepath"

	"github.com/chuccp/go-ai-agent/internal/agent/tool"
	"github.com/chuccp/go-ai-agent/internal/ai"
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/rest"
	"github.com/chuccp/go-ai-agent/internal/runner"
	"github.com/chuccp/go-ai-agent/internal/service"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/component/cache"
	"github.com/chuccp/go-web-frame/component/cors"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

const configFilePath = "application.yml"

// getDataDir returns the absolute path to the data directory.
// For desktop mode, it uses the executable's directory.
// For web mode, it uses the current working directory.
func getDataDir(webMode bool) string {
	if webMode {
		// Web mode: use current working directory
		return "./data"
	}
	
	// Desktop mode: use executable's directory
	execPath, err := os.Executable()
	if err != nil {
		log.Warn("Failed to get executable path, using current directory", zap.Error(err))
		return "./data"
	}
	
	return filepath.Join(filepath.Dir(execPath), "data")
}

// Create builds the web framework with all services and REST endpoints.
// webMode=true skips desktop-specific services.
func Create(webMode bool) *wf.WebFrame {
	loadConfig, isFirstRun := loadOrCreateConfig(webMode)

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
	}

	if isFirstRun {
		log.Info("First run detected, enabling setup wizard", zap.String("configPath", configFilePath))
		rests = append(rests, rest.NewSetupRest(configFilePath))
	}
	rests = append(rests,
		rest.NewFlowRest(),
		rest.NewModelRest(),
	)

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
// Also returns the ChatRunner for Wails IPC binding.
func CreateDesktop() (*wf.WebFrame, *runner.ChatRunner) {
	loadConfig, isFirstRun := loadOrCreateConfig(false)

	builder := wf.NewBuilder(loadConfig)

	services := []core.IService{
		&cache.Cache{},
		chat.NewDefaultChatService(),
		ai.NewGenService(),
		tool.NewRegistry(),
		&DesktopInitService{},
		&service.FlowService{},
	}

	builder.Service(services...)

	chatRunner := runner.NewChatRunner()
	builder.Runner(chatRunner)
	builder.Runner(runner.NewFlowRunner())

	rests := []core.IRest{
		&rest.Api{},
		rest.NewChatRest(),
	}

	if isFirstRun {
		log.Info("First run detected, enabling setup wizard", zap.String("configPath", configFilePath))
		rests = append(rests, rest.NewSetupRest(configFilePath))
	}
	rests = append(rests,
		rest.NewFlowRest(),
		rest.NewModelRest(),
	)

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
	return builder.Build(), chatRunner
}

func loadOrCreateConfig(webMode bool) (*config.Config, bool) {
	loadConfig, err := config.LoadConfig(configFilePath)
	if err == nil {
		// Desktop mode: always ensure desktop flag is set
		if !webMode {
			loadConfig.Put("system.desktop", true)
		}
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
		// Get absolute data directory path based on executable location
		dataDir := getDataDir(webMode)
		
		// Create data directory if it doesn't exist
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Warn("Failed to create data directory", zap.Error(err))
		}
		
		dbPath := filepath.Join(dataDir, "go-ai-agent.db")
		cfg.Put("web.db.type", "sqlite")
		cfg.Put("web.db.path", dbPath)
		cfg.Put("system.desktop", true)
	}

	return cfg, true
}
