package gemini

type GeminiConfig struct {
	APIKey string `mapstructure:"apiKey" yaml:"apiKey" json:"apiKey"`
	Model  string `mapstructure:"model" yaml:"model" json:"model"`
}

func (c *GeminiConfig) GetModel() string {
	if c.Model == "" {
		return "gemini-2.5-flash"
	}
	return c.Model
}
