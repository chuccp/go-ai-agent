package entity

import "time"

// AIModel AI 模型配置
type AIModel struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128" json:"name"`                // 显示名称
	Provider    string    `gorm:"size:64;index" json:"provider"`       // 提供商: openai, anthropic, gemini, deepseek, volcengine, openai_compat
	Model       string    `gorm:"size:128" json:"model"`               // 模型标识: gpt-4o, claude-sonnet-4-20250514 等
	Category    string    `gorm:"size:64;index" json:"category"`       // 分类: llm, image, voice, video
	APIKey      string    `gorm:"size:512" json:"api_key,omitempty"`   // API Key (可选，覆盖全局配置)
	BaseURL     string    `gorm:"size:512" json:"base_url,omitempty"`  // 自定义 BaseURL (可选)
	IsDefault   bool      `gorm:"default:false" json:"is_default"`     // 是否为该分类的默认模型
	IsBase      bool      `gorm:"default:false" json:"is_base"`       // 是否为基础模型（首次初始化时配置）
	Description string    `gorm:"size:512" json:"description"`         // 描述
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AIModel) TableName() string {
	return "ai_models"
}
