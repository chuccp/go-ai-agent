package server

const configKey = "api.chat"

// ChatConfig mirrors the CLI's config structure for LLM provider registration.
type ChatConfig struct {
	Name    string
	Type    string
	BaseUrl string
	ApiKey  string
	Model   string
	Default bool
}
