package config

import (
	"os"

	"github.com/chuccp/go-web-frame/config"
	"gopkg.in/yaml.v3"
)

const ConfigFilePath = "application.yml"

// ApplicationConfig mirrors the full application.yml structure for marshaling.
// yaml tags must match the existing camelCase keys exactly.
type ApplicationConfig struct {
	System SystemConfig `yaml:"system"`
	Web    WebConfig    `yaml:"web"`
	Flow   FlowConfig   `yaml:"flow"`
}

type SystemConfig struct {
	ApiPrefix string `yaml:"apiPrefix"`
	Debug     bool   `yaml:"debug"`
	Init      bool   `yaml:"init"`
}

type WebConfig struct {
	DB     DBConfig     `yaml:"db"`
	Server ServerConfig `yaml:"server"`
}

type DBConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path,omitempty"`
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	Database string `yaml:"database,omitempty"`
	Charset  string `yaml:"charset,omitempty"`
	SSLMode  string `yaml:"sslMode,omitempty"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type FlowCacheConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type FlowConfig struct {
	Cache          FlowCacheConfig `yaml:"cache"`
	MaxConcurrency int             `yaml:"maxConcurrency"`
	AppsPath       string          `yaml:"appsPath"`
}

// DefaultAppConfig returns a set of defaults matching the existing application.yml.
func DefaultAppConfig() *ApplicationConfig {
	return &ApplicationConfig{
		System: SystemConfig{
			ApiPrefix: "/api",
			Debug:     true,
			Init:      false,
		},
		Web: WebConfig{
			Server: ServerConfig{
				Port: 19009,
			},
		},
		Flow: FlowConfig{
			Cache: FlowCacheConfig{
				Enabled: true,
				Path:    "./data/cache",
			},
			MaxConcurrency: 4,
			AppsPath:       "./data/apps",
		},
	}
}

// WriteAppConfig marshals the config to YAML and writes it to ConfigFilePath.
func WriteAppConfig(cfg *ApplicationConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigFilePath, data, 0644)
}

// BuildAppConfigFromRuntime reads all keys from the runtime viper config into the typed struct.
func BuildAppConfigFromRuntime(cfg config.IConfig) *ApplicationConfig {
	appCfg := DefaultAppConfig()

	// System
	if cfg.HasKey("system.apiPrefix") || cfg.HasKey("system.apiprefix") {
		appCfg.System.ApiPrefix = cfg.GetStringOrDefault("system.apiPrefix", "/api")
		if appCfg.System.ApiPrefix == "" {
			appCfg.System.ApiPrefix = cfg.GetStringOrDefault("system.apiprefix", "/api")
		}
	}
	if cfg.HasKey("system.debug") {
		appCfg.System.Debug = cfg.GetBoolOrDefault("system.debug", true)
	}
	if cfg.HasKey("system.init") {
		appCfg.System.Init = cfg.GetBoolOrDefault("system.init", false)
	}

	// Web DB
	if cfg.HasKey("web.db") || cfg.HasKey("web.db.type") {
		appCfg.Web.DB.Type = cfg.GetString("web.db.type")
		appCfg.Web.DB.Path = cfg.GetString("web.db.path")
		appCfg.Web.DB.Host = cfg.GetString("web.db.host")
		appCfg.Web.DB.Port = cfg.GetInt("web.db.port")
		appCfg.Web.DB.Username = cfg.GetString("web.db.username")
		appCfg.Web.DB.Password = cfg.GetString("web.db.password")
		appCfg.Web.DB.Database = cfg.GetString("web.db.database")
		if appCfg.Web.DB.Database == "" {
			appCfg.Web.DB.Database = cfg.GetString("web.db.dbname")
		}
		appCfg.Web.DB.Charset = cfg.GetString("web.db.charset")
		appCfg.Web.DB.SSLMode = cfg.GetString("web.db.sslMode")
		if appCfg.Web.DB.SSLMode == "" {
			appCfg.Web.DB.SSLMode = cfg.GetString("web.db.sslmode")
		}
	}

	// Web Server
	if cfg.HasKey("web.server.port") {
		appCfg.Web.Server.Port = cfg.GetInt("web.server.port")
	}

	return appCfg
}
