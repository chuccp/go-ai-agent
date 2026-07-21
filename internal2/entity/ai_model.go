package entity

import "time"

// AIModel AI model configuration
type AIModel struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128" json:"name"`                // Display name
	Provider    string    `gorm:"size:64;index" json:"provider"`       // Provider: openai, anthropic, gemini, deepseek, volcengine, openai_compat
	Model       string    `gorm:"size:128" json:"model"`               // Model identifier: gpt-4o, claude-sonnet-4-20250514 , etc.
	Category    string    `gorm:"size:64;index" json:"category"`       // Category: llm, image, voice, video
	APIKey      string    `gorm:"size:512" json:"api_key,omitempty"`   // API Key (optional, overrides global config)
	BaseURL     string    `gorm:"size:512" json:"base_url,omitempty"`  // Custom BaseURL (optional)
	IsDefault   bool      `gorm:"default:false" json:"is_default"`     // Whether this is the default model for the category
	IsBase      bool      `gorm:"default:false" json:"is_base"`       // Whether this is a base model (configured during initial setup)
	Description string    `gorm:"size:512" json:"description"`         // Description
	InputTypes         string    `gorm:"size:256" json:"input_types,omitempty"`          // Supported input types: text,image,audio (comma-separated)
	OutputTypes        string    `gorm:"size:256" json:"output_types,omitempty"`         // Supported output types: text,image,audio (comma-separated)
	SupportsMultimodal bool      `gorm:"default:false" json:"supports_multimodal"`       // Whether multimodal input (images) is supported
	ThinkingLevel      string    `gorm:"size:16;default:off" json:"thinking_level"`       // Thinking level: off, low, medium, high, max
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AIModel) TableName() string {
	return "ai_models"
}
