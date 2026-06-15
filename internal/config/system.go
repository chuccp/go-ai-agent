package config

const SYSTEM = "system"

type System struct {
	ApiPrefix string `mapstructure:"apiPrefix"`
	Debug     bool   `mapstructure:"debug"`
}
