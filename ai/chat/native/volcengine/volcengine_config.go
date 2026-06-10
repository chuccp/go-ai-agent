package volcengine

// VolcengineConfig 火山引擎配置
type VolcengineConfig struct {
	APIKey string `mapstructure:"apiKey" yaml:"apiKey" json:"apiKey"`
	Model  string `mapstructure:"model" yaml:"model" json:"model"`
}

func (c *VolcengineConfig) GetModel() string {
	if c.Model == "" {
		return "doubao-seed-1-6"
	}
	return c.Model
}
