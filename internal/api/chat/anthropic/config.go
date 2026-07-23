package anthropic

type Config struct {
	BaseURL string `mapstructure:"baseUrl" yaml:"baseUrl" json:"baseUrl"`
	APIKey  string `mapstructure:"apiKey" yaml:"apiKey" json:"apiKey"`
	Model   string `mapstructure:"model" yaml:"model" json:"model"`
}
