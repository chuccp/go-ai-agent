package rest

import (
	"errors"

	"github.com/chuccp/go-ai-agent/internal/ai/chat"
	aiTypes "github.com/chuccp/go-ai-agent/internal/ai/types"
	appconfig "github.com/chuccp/go-ai-agent/internal/config"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/util"
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

	// Setup status (per-step progress)
	// Complete setup
	ctx.Post("/api/setup/complete", s.completeSetup)

	log.Info("Setup REST initialized")
	return nil
}

// ---- Step 1: Database ----

func (s *SetupRest) putDbInit(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("system is already initialized, cannot reconfigure")
	}

	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	dbType := j.GetString("type")
	if dbType == "" {
		return nil, errors.New("please select a database type (sqlite / mysql / postgresql)")
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
		return nil, errors.New("unsupported database type: " + dbType)
	}

	// Create DB connection via framework
	createdDB, err := db.CreateDB(cfg)
	if err != nil {
		return nil, errors.New("database connection failed: " + err.Error())
	}

	// Switch all models to the new DB (auto-migrates tables)
	err = s.context.DefaultModelGroup().SwitchDB(createdDB, s.context)
	if err != nil {
		return nil, errors.New("database initialization failed: " + err.Error())
	}

	log.Info("Database configured", zap.String("type", dbType))
	return web.Ok("Database configured successfully"), nil
}

func (s *SetupRest) testConnection(req *web.Request) (any, error) {
	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	dbType := j.GetString("type")
	if dbType == "" {
		return nil, errors.New("please select a database type")
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
		return nil, errors.New("unsupported database type: " + dbType)
	}

	// Validate by creating a test connection
	_, err = db.CreateDB(tmpCfg)
	if err != nil {
		return nil, errors.New("connection test failed: " + err.Error())
	}

	return web.Ok("Connection test successful"), nil
}

// ---- Step 2: Admin Account ----

func (s *SetupRest) putAdminInit(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("system is already initialized, cannot reconfigure")
	}

	// Guard: DB must be configured
	if !cfg.HasKey("web.db") || cfg.GetString("web.db.type") == "" {
		return nil, errors.New("please complete database configuration first")
	}

	j, err := req.Json()
	if err != nil {
		return nil, err
	}

	username := j.GetString("username")
	password := j.GetString("password")

	if username == "" || password == "" {
		return nil, errors.New("username and password cannot be empty")
	}

	hash, err := util.HashPassword(password)
	if err != nil {
		return nil, err
	}

	adminModel := core.GetModel[*model.AdminUserModel](s.context)
	if adminModel == nil {
		return nil, errors.New("admin model not initialized")
	}

	hasAdmin, err := adminModel.WithContext(req.Ctx()).HasAdminUser()
	if err != nil {
		return nil, errors.New("failed to query admin status: " + err.Error())
	}

	if hasAdmin {
		// Reset existing admin password
		user, err := adminModel.WithContext(req.Ctx()).FindByUsername(username)
		if err != nil {
			return nil, errors.New("admin user not found, please check username")
		}
		if err := adminModel.WithContext(req.Ctx()).UpdatePassword(user.Id, hash); err != nil {
			return nil, errors.New("failed to reset password: " + err.Error())
		}
		log.Info("Admin password reset", zap.String("username", username))
	} else {
		// Create new admin
		if err := adminModel.WithContext(req.Ctx()).Create(&entity.AdminUser{
			Username:     username,
			PasswordHash: hash,
			IsAdmin:      true,
		}); err != nil {
			return nil, errors.New("failed to create admin: " + err.Error())
		}
		log.Info("Admin account created", zap.String("username", username))
	}

	return web.Ok("Admin account configured successfully"), nil
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

	hasAdmin, _ := adminModel.WithContext(req.Ctx()).HasAdminUser()
	adminName := ""
	if hasAdmin {
		user, err := adminModel.WithContext(req.Ctx()).FindByUsername("")
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
		return nil, errors.New("system is already initialized, cannot reconfigure")
	}

	// Guard: DB must be configured
	if !cfg.HasKey("web.db") || cfg.GetString("web.db.type") == "" {
		return nil, errors.New("please complete database configuration first")
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
		return nil, errors.New("provider and model identifier cannot be empty")
	}
	if category == "" {
		category = aiTypes.CategoryLLM
	}
	if name == "" {
		name = provider + " " + modelName
	}

	aiModel := core.GetModel[*model.AIModelModel](s.context)
	if aiModel == nil {
		return nil, errors.New("AI model not initialized")
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

	if err := aiModel.WithContext(req.Ctx()).Create(m); err != nil {
		return nil, errors.New("failed to create base model: " + err.Error())
	}

	log.Info("Base model configured", zap.String("provider", provider), zap.String("model", modelName))
	return web.Data(m), nil
}

// ---- Provider Defaults ----

func (s *SetupRest) getProviders(_ *web.Request) (any, error) {
	return web.Data(chat.GetGroupedProviderInfo()), nil
}

// ---- Setup Status ----

func (s *SetupRest) getSetupStatus(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	initialized := cfg.GetBoolOrDefault("system.init", false)
	dbConfigured := cfg.HasKey("web.db") && cfg.GetString("web.db.type") != ""

	adminConfigured := false
	if dbConfigured {
		adminModel := core.GetModel[*model.AdminUserModel](s.context)
		if adminModel != nil {
			hasAdmin, _ := adminModel.WithContext(req.Ctx()).HasAdminUser()
			adminConfigured = hasAdmin
		}
	}

	mode := "web"
	if cfg.GetBoolOrDefault("system.desktop", false) {
		mode = "desktop"
	}

	return web.Data(map[string]interface{}{
		"initialized":      initialized,
		"db_configured":    dbConfigured,
		"admin_configured": adminConfigured,
		"mode":             mode,
	}), nil
}

// ---- Complete Setup ----

func (s *SetupRest) completeSetup(req *web.Request) (any, error) {
	cfg := s.context.GetConfig()

	// Guard: already fully initialized
	if cfg.GetBoolOrDefault("system.init", false) {
		return nil, errors.New("system has already been initialized")
	}

	// Mark as initialized
	cfg.Put("system.init", true)

	// Build the full ApplicationConfig from runtime config
	appCfg := appconfig.BuildAppConfigFromRuntime(cfg)

	// Write to application.yml
	if err := appconfig.WriteAppConfig(appCfg); err != nil {
		return nil, errors.New("failed to write config file: " + err.Error())
	}

	log.Info("System initialization complete, config file written")
	return web.Ok("System initialization complete"), nil
}
