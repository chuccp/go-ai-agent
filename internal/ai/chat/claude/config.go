package claude

// Config Anthropic Messages API compatible config
type Config struct {
	BaseURL string `mapstructure:"baseUrl" yaml:"baseUrl" json:"baseUrl"`
	APIKey  string `mapstructure:"apiKey" yaml:"apiKey" json:"apiKey"`
	Model   string `mapstructure:"model" yaml:"model" json:"model"`
}

func (c *Config) GetBaseURL() string { return c.BaseURL }
func (c *Config) GetModel() string   { return c.Model }
