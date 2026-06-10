package rest

import (
	"errors"

	"github.com/chuccp/go-ai-agent/ai/chat"
	aiTypes "github.com/chuccp/go-ai-agent/ai/types"
	appconfig "github.com/chuccp/go-ai-agent/config"
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/util"
	"github.com/chuccp/go-web-frame/config"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
	"go.uber.org/zap"
)

// SetupRest handles the first-run setup wizard endpoints.
// These endpoints are only registered when the system is not yet initialized.
type SetupRest struct {
	context    *core.Context
	configPath string
}

// NewSetupRest creates a new setup REST handler.
func NewSetupRest(configPath string) *SetupRest {
	return &SetupRest{configPath: configPath}
}

// Init registers all setup endpoints.
func (s *SetupRest) Init(ctx *core.Context) error {
	s.context = ctx

	// Step 1: Database
	ctx.Put("/api/setup/db", s.putDbInit)
	ctx.Post("/api/setup/db/test", s.testConnection)

	// Step 2: Admin account
	ctx.Put("/api/setup/admin", s.putAdminInit)
	ctx.Get("/api/setup/admin/exists", s.getAdminExists)

	// Step 3: Base LLM model
	ctx.Put("/api/setup/model", s.putModelInit)

	// Provider defaults for the setup wizard
	ctx.Get("/api/setup/providers", s.getProviders)

	// Complete setup
	ctx.Post("/api/setup/complete", s.completeSetup)

	log.Info("Setup REST 已初始化")
	return nil
}

// ---- Step 1: Database ----

func (s *SetupRest) putDbInit(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("系统已初始化，无法重新配置")
	}

	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	dbType := j.GetString("type")
	if dbType == "" {
		return nil, errors.New("请选择数据库类型 (sqlite / mysql / postgresql)")
	}

	// Build web.db config from request
	switch dbType {
	case "sqlite":
		path := j.GetString("path")
		if path == "" {
			path = "./data/go-ai-agent.db"
		}
		cfg.Put("web.db.type", "sqlite")
		cfg.Put("web.db.path", path)
	case "mysql":
		cfg.Put("web.db.type", "mysql")
		cfg.Put("web.db.host", j.GetString("host"))
		cfg.Put("web.db.port", j.GetInt("port"))
		cfg.Put("web.db.username", j.GetString("username"))
		cfg.Put("web.db.password", j.GetString("password"))
		cfg.Put("web.db.database", j.GetString("database"))
		if j.GetString("charset") != "" {
			cfg.Put("web.db.charset", j.GetString("charset"))
		}
	case "postgresql", "postgres":
		cfg.Put("web.db.type", "postgres")
		cfg.Put("web.db.host", j.GetString("host"))
		cfg.Put("web.db.port", j.GetInt("port"))
		cfg.Put("web.db.username", j.GetString("username"))
		cfg.Put("web.db.password", j.GetString("password"))
		cfg.Put("web.db.database", j.GetString("database"))
		if j.GetString("sslMode") != "" {
			cfg.Put("web.db.sslMode", j.GetString("sslMode"))
		}
	default:
		return nil, errors.New("不支持的数据库类型: " + dbType)
	}

	// Create DB connection via framework
	createdDB, err := db.CreateDB(cfg)
	if err != nil {
		return nil, errors.New("数据库连接失败: " + err.Error())
	}

	// Switch all models to the new DB (auto-migrates tables)
	err = s.context.DefaultModelGroup().SwitchDB(createdDB, s.context)
	if err != nil {
		return nil, errors.New("数据库初始化失败: " + err.Error())
	}

	log.Info("数据库配置完成", zap.String("type", dbType))
	return web.Ok("数据库配置成功"), nil
}

func (s *SetupRest) testConnection(req *web.Request) (any, error) {
	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	dbType := j.GetString("type")
	if dbType == "" {
		return nil, errors.New("请选择数据库类型")
	}

	// Build a temporary in-memory config for testing
	tmpCfg := config.NewConfig()

	switch dbType {
	case "sqlite":
		path := j.GetString("path")
		if path == "" {
			path = "./data/go-ai-agent.db"
		}
		tmpCfg.Put("web.db.type", "sqlite")
		tmpCfg.Put("web.db.path", path)
	case "mysql":
		tmpCfg.Put("web.db.type", "mysql")
		tmpCfg.Put("web.db.host", j.GetString("host"))
		tmpCfg.Put("web.db.port", j.GetInt("port"))
		tmpCfg.Put("web.db.username", j.GetString("username"))
		tmpCfg.Put("web.db.password", j.GetString("password"))
		tmpCfg.Put("web.db.database", j.GetString("database"))
		if j.GetString("charset") != "" {
			tmpCfg.Put("web.db.charset", j.GetString("charset"))
		}
	case "postgresql", "postgres":
		tmpCfg.Put("web.db.type", "postgres")
		tmpCfg.Put("web.db.host", j.GetString("host"))
		tmpCfg.Put("web.db.port", j.GetInt("port"))
		tmpCfg.Put("web.db.username", j.GetString("username"))
		tmpCfg.Put("web.db.password", j.GetString("password"))
		tmpCfg.Put("web.db.database", j.GetString("database"))
		if j.GetString("sslMode") != "" {
			tmpCfg.Put("web.db.sslMode", j.GetString("sslMode"))
		}
	default:
		return nil, errors.New("不支持的数据库类型: " + dbType)
	}

	// Validate by creating a test connection
	_, err = db.CreateDB(tmpCfg)
	if err != nil {
		return nil, errors.New("连接测试失败: " + err.Error())
	}

	return web.Ok("连接测试成功"), nil
}

// ---- Step 2: Admin Account ----

func (s *SetupRest) putAdminInit(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("系统已初始化，无法重新配置")
	}

	// Guard: DB must be configured
	if !cfg.HasKey("web.db") || cfg.GetString("web.db.type") == "" {
		return nil, errors.New("请先完成数据库配置")
	}

	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	username := j.GetString("username")
	password := j.GetString("password")

	if username == "" || password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}

	hash, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}

	adminModel := core.GetModel[*model.AdminUserModel](s.context)
	if adminModel == nil {
		return nil, errors.New("管理员模型未初始化")
	}

	hasAdmin, err := adminModel.HasAdminUser()
	if err != nil {
		return nil, errors.New("查询管理员状态失败: " + err.Error())
	}

	if hasAdmin {
		// Reset existing admin password
		user, err := adminModel.FindByUsername(username)
		if err != nil {
			return nil, errors.New("管理员用户不存在，请检查用户名")
		}
		if err := adminModel.UpdatePassword(user.Id, hash); err != nil {
			return nil, errors.New("重置密码失败: " + err.Error())
		}
		log.Info("管理员密码已重置", zap.String("username", username))
	} else {
		// Create new admin
		if err := adminModel.Create(&entity.AdminUser{
			Username:     username,
			PasswordHash: hash,
			IsAdmin:      true,
		}); err != nil {
			return nil, errors.New("创建管理员失败: " + err.Error())
		}
		log.Info("管理员账号已创建", zap.String("username", username))
	}

	return web.Ok("管理员账号配置成功"), nil
}

func (s *SetupRest) getAdminExists(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()
	if !cfg.HasKey("web.db") || cfg.GetString("web.db.type") == "" {
		return web.Data(map[string]interface{}{
			"hasAdmin":  false,
			"adminName": "",
		}), nil
	}

	adminModel := core.GetModel[*model.AdminUserModel](s.context)
	if adminModel == nil {
		return web.Data(map[string]interface{}{
			"hasAdmin":  false,
			"adminName": "",
		}), nil
	}

	hasAdmin, _ := adminModel.HasAdminUser()
	adminName := ""
	if hasAdmin {
		user, err := adminModel.FindByUsername("")
		if err == nil && user != nil {
			adminName = user.Username
		}
	}

	return web.Data(map[string]interface{}{
		"hasAdmin":  hasAdmin,
		"adminName": adminName,
	}), nil
}

// ---- Step 3: Base LLM Model ----

func (s *SetupRest) putModelInit(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("系统已初始化，无法重新配置")
	}

	// Guard: DB must be configured
	if !cfg.HasKey("web.db") || cfg.GetString("web.db.type") == "" {
		return nil, errors.New("请先完成数据库配置")
	}

	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	name := j.GetString("name")
	provider := j.GetString("provider")
	modelName := j.GetString("model")
	category := j.GetString("category")

	if provider == "" || modelName == "" {
		return nil, errors.New("提供商和模型标识不能为空")
	}
	if category == "" {
		category = aiTypes.CategoryLLM
	}
	if name == "" {
		name = provider + " " + modelName
	}

	aiModel := core.GetModel[*model.AIModelModel](s.context)
	if aiModel == nil {
		return nil, errors.New("AI 模型未初始化")
	}

	m := &entity.AIModel{
		Name:        name,
		Provider:    provider,
		Model:       modelName,
		Category:    category,
		APIKey:      j.GetString("api_key"),
		BaseURL:     j.GetString("base_url"),
		IsDefault:   true,
		IsBase:      true,
		Description: j.GetString("description"),
	}

	if err := aiModel.Create(m); err != nil {
		return nil, errors.New("创建基础模型失败: " + err.Error())
	}

	log.Info("基础模型已配置", zap.String("provider", provider), zap.String("model", modelName))
	return web.Data(m), nil
}

// ---- Provider Defaults ----

func (s *SetupRest) getProviders(_ *web.Request) (any, error) {
	return web.Data(chat.GetGroupedProviderInfo()), nil
}

// ---- Complete Setup ----

func (s *SetupRest) completeSetup(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("系统已完成初始化")
	}

	// Mark as initialized
	cfg.Put("system.init", true)

	// Build the full ApplicationConfig from runtime config
	appCfg := appconfig.BuildAppConfigFromRuntime(cfg)

	// Write to application.yml
	if err := appconfig.WriteAppConfig(appCfg); err != nil {
		return nil, errors.New("写入配置文件失败: " + err.Error())
	}

	log.Info("系统初始化完成，配置文件已写入")
	return web.Ok("系统初始化完成"), nil
}
