package app

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-ai-agent/model"
	"github.com/chuccp/go-ai-agent/util"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"go.uber.org/zap"
)

// DesktopInitService auto-configures SQLite database and default admin account
// during the framework initialization.
type DesktopInitService struct{}

func (s *DesktopInitService) Init(ctx *core.Context) error {
	cfg := ctx.GetConfig()

	if cfg.GetBoolOrDefault("system.init", false) {
		return nil
	}
	if !cfg.GetBoolOrDefault("system.desktop", false) {
		return nil
	}

	log.Info("[desktop] auto-initializing desktop mode (SQLite + default admin)")

	adminModel := core.GetModel[*model.AdminUserModel](ctx)
	if adminModel != nil {
		hasAdmin, err := adminModel.HasAdminUser()
		if err != nil {
			log.Warn("[desktop] failed to check admin user", zap.Error(err))
		} else if !hasAdmin {
			hash, err := util.HashPassword("admin")
			if err != nil {
				return err
			}
			if err := adminModel.Create(&entity.AdminUser{
				Username:     "admin",
				PasswordHash: hash,
				IsAdmin:      true,
			}); err != nil {
				log.Warn("[desktop] failed to create default admin", zap.Error(err))
			} else {
				log.Info("[desktop] default admin account created (admin/admin)")
			}
		}
	}

	return nil
}
