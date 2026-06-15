package app

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal/agent/tool"
	"github.com/chuccp/go-ai-agent/internal/ai"
	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	"github.com/chuccp/go-ai-agent/internal/ai/chat/common"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/rest"
	"github.com/chuccp/go-ai-agent/internal/runner"
	"github.com/chuccp/go-ai-agent/internal/service"
	"github.com/chuccp/go-ai-agent/internal/skill"
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
		ai.NewGenService(),
		tool.NewRegistry(),
		&service.FlowService{},
		skill.NewService(),
		&skillExecutorWire{},
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
		rest.NewSkillRest(),
		rest.NewPackageRest(),
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
		&model.PackageModel{},
		&model.PackageResourceModel{},
		&model.PackageConfigModel{},
		&model.SkillModel{},
		&model.SkillPromptModel{},
		&model.SkillResourceModel{},
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
		skill.NewService(),
		&skillExecutorWire{},
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
	} else {
		rests = append(rests,
			rest.NewFlowRest(),
			rest.NewModelRest(),
			rest.NewSkillRest(),
			rest.NewPackageRest(),
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
		&model.PackageModel{},
		&model.PackageResourceModel{},
		&model.PackageConfigModel{},
		&model.SkillModel{},
		&model.SkillPromptModel{},
		&model.SkillResourceModel{},
	)
	builder.Filter(cors.NewCrosFilter())
	return builder.Build(), chatRunner
}

// skillExecutorWire injects the chat service as the skill executor after both
// skill.Service and chat.UnifiedChatService have been initialized.
type skillExecutorWire struct{}

func (w *skillExecutorWire) Init(ctx *core.Context) error {
	svc := core.GetService[*skill.Service](ctx)
	chatSvc := core.GetService[*chat.UnifiedChatService](ctx)
	if svc != nil && chatSvc != nil {
		svc.SetExecutor(&skillChatExecutor{chat: chatSvc})
		svc.SetDefaultModelPath(chatSvc.GetDefaultPath())
	}
	return nil
}

type skillChatExecutor struct {
	chat *chat.UnifiedChatService
}

func (e *skillChatExecutor) Execute(ctx context.Context, modelPath, prompt string) (string, error) {
	return e.chat.ChatWithContext(ctx, modelPath, prompt, &common.LLMOptions{})
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
		cfg.Put("web.db.type", "sqlite")
		cfg.Put("web.db.path", "./data/go-ai-agent.db")
		cfg.Put("system.desktop", true)
	}

	return cfg, true
}
