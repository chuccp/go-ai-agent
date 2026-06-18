package rest

import (
	"errors"
	"os"
	"path/filepath"

	appconfig "github.com/chuccp/go-ai-agent/internal/config"
	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-ai-agent/internal/model"
	"github.com/chuccp/go-ai-agent/internal/runner"
	"github.com/chuccp/go-ai-agent/internal/util"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/log"
	"github.com/chuccp/go-web-frame/web"
	"go.uber.org/zap"
)

// SystemRest handles system-level operations like clearing data.
// Always registered regardless of initialization state.
type SystemRest struct {
	context *core.Context
}

func (s *SystemRest) Init(ctx *core.Context) error {
	s.context = ctx
	ctx.Post("/api/system/clear-db", s.clearDatabase)
	ctx.Post("/api/system/clear-all", s.clearAll)
	log.Info("System REST initialized")
	return nil
}

// clearDatabase drops all tables and recreates them empty, then sets
// system.init = false so the app redirects back to the Setup Wizard.
func (s *SystemRest) clearDatabase(req *web.Request) (any, error) {
	if err := s.performClear(false); err != nil {
		return nil, err
	}
	return web.Ok("Database cleared"), nil
}

// clearAll does everything clearDatabase does, plus removes all app data
// directories, uploads, and cache.
func (s *SystemRest) clearAll(req *web.Request) (any, error) {
	if err := s.performClear(true); err != nil {
		return nil, err
	}
	return web.Ok("All data cleared"), nil
}

// performClear is the shared logic for clearDatabase and clearAll.
// If clearAppData is true, also removes app directories, uploads, cache, etc.
func (s *SystemRest) performClear(clearAppData bool) error {
	mg := s.context.DefaultModelGroup()
	if mg == nil {
		return errors.New("no model group available")
	}

	// Drop and recreate all tables
	models := mg.GetModel()
	for _, m := range models {
		if err := m.DeleteTable(); err != nil {
			log.Warn("failed to drop table", zap.String("table", m.GetTableName()), zap.Error(err))
		}
	}
	for _, m := range models {
		if err := m.CreateTable(); err != nil {
			log.Warn("failed to recreate table", zap.String("table", m.GetTableName()), zap.Error(err))
		}
	}
	log.Info("all tables dropped and recreated", zap.Int("count", len(models)))

	// Set system.init = false and persist config
	cfg := s.context.GetConfig()
	cfg.Put("system.init", false)
	appCfg := appconfig.BuildAppConfigFromRuntime(cfg)
	if err := appconfig.WriteAppConfig(appCfg); err != nil {
		return errors.New("failed to write config file: " + err.Error())
	}

	// Reset ChatRunner providers so they reload from the (now empty) DB
	chatRunner := core.GetRunner[*runner.ChatRunner](s.context)
	if chatRunner != nil {
		chatRunner.ResetProviders()
	}

	// In desktop mode, auto-create the default admin user (same as DesktopInitService)
	// so the setup wizard skips the admin step and goes straight to model configuration.
	if cfg.GetBoolOrDefault("system.desktop", false) {
		s.autoCreateDesktopAdmin()
	}

	// Optionally clear app data directories
	if clearAppData {
		appsPath := cfg.GetStringOrDefault("flow.appsPath", "./data/apps")
		if appsPath != "" {
			if err := clearDirContents(appsPath); err != nil {
				log.Warn("failed to clear apps directory", zap.String("path", appsPath), zap.Error(err))
			}
		}
		if err := clearDirContents("./data/uploads"); err != nil {
			log.Warn("failed to clear uploads directory", zap.Error(err))
		}
		if err := clearDirContents("./data/cache"); err != nil {
			log.Warn("failed to clear cache directory", zap.Error(err))
		}
		if err := clearDirContents("./data/gen"); err != nil {
			log.Warn("failed to clear gen directory", zap.Error(err))
		}
		log.Info("app data directories cleared")
	}

	log.Info("clear complete", zap.Bool("clearAppData", clearAppData))
	return nil
}

// autoCreateDesktopAdmin creates the default admin/admin account if it doesn't exist.
// This mirrors the logic in DesktopInitService.Init() so that after a database
// clear in desktop mode, the setup wizard can skip the admin step.
func (s *SystemRest) autoCreateDesktopAdmin() {
	adminModel := core.GetModel[*model.AdminUserModel](s.context)
	if adminModel == nil {
		return
	}
	hasAdmin, err := adminModel.HasAdminUser()
	if err != nil {
		log.Warn("[system] failed to check admin user after clear", zap.Error(err))
		return
	}
	if hasAdmin {
		return
	}
	hash, err := util.HashPassword("admin")
	if err != nil {
		log.Warn("[system] failed to hash default admin password", zap.Error(err))
		return
	}
	if err := adminModel.Create(&entity.AdminUser{
		Username:     "admin",
		PasswordHash: hash,
		IsAdmin:      true,
	}); err != nil {
		log.Warn("[system] failed to create default admin after clear", zap.Error(err))
	} else {
		log.Info("[system] default admin account created (admin/admin)")
	}
}

// clearDirContents removes all files and subdirectories inside the given
// directory, but keeps the directory itself.
func clearDirContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist, nothing to clear
		}
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			log.Warn("failed to remove", zap.String("path", path), zap.Error(err))
		}
	}
	return nil
}
