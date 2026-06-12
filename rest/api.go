package rest

import (
	"github.com/chuccp/go-ai-agent/config"
	"github.com/chuccp/go-ai-agent/model"
	wf "github.com/chuccp/go-web-frame"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/web"
)

type Api struct {
	context *core.Context
	system  *config.System
}

func (l *Api) Init(context *core.Context) error {
	l.context = context
	var err error
	l.system, err = wf.UnmarshalKeyConfig[*config.System](config.SYSTEM, l.context)
	if err != nil {
		return err
	}

	l.context.Get("/", l.index)
	l.context.Get("/api/health", l.health)
	l.context.Get("/api/setup/status", l.getSetupStatus)
	return nil
}

func (l *Api) index(request *web.Request) (any, error) {
	return map[string]interface{}{
		"name":    "go-ai-agent",
		"version": "1.0.0",
	}, nil
}

func (l *Api) health(request *web.Request) (any, error) {
	return web.Ok("healthy"), nil
}

func (l *Api) getSetupStatus(_ *web.Request) (any, error) {
	cfg := l.context.GetConfig()
	initialized := cfg.GetBoolOrDefault("system.init", false)
	dbConfigured := cfg.HasKey("web.db") && cfg.GetString("web.db.type") != ""

	hasAdmin := false
	hasBaseModel := false

	if dbConfigured {
		adminModel := core.GetModel[*model.AdminUserModel](l.context)
		if adminModel != nil {
			ok, _ := adminModel.HasAdminUser()
			hasAdmin = ok
		}

		aiModel := core.GetModel[*model.AIModelModel](l.context)
		if aiModel != nil {
			models, _ := aiModel.FindBase()
			hasBaseModel = len(models) > 0
		}
	}

	mode := "web"
	if cfg.GetBoolOrDefault("system.desktop", false) {
		mode = "desktop"
	}

	return web.Data(map[string]interface{}{
		"initialized":      initialized,
		"db_configured":    dbConfigured,
		"admin_configured": hasAdmin,
		"hasBaseModel":     hasBaseModel,
		"mode":             mode,
	}), nil
}
