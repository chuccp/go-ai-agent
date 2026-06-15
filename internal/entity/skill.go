package entity

import "time"

// Skill is a reusable capability bundled in a package; it can be used as a flow node or called standalone.
type Skill struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	SkillId     string    `gorm:"size:128;uniqueIndex" json:"skill_id"` // identifier from skill.json
	PackageId   uint      `gorm:"index" json:"package_id,omitempty"`
	Name        string    `gorm:"size:256" json:"name"`
	Version     string    `gorm:"size:64" json:"version"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"size:512" json:"icon,omitempty"`
	Inputs      string    `gorm:"type:text" json:"inputs,omitempty"`   // JSON schema
	Outputs     string    `gorm:"type:text" json:"outputs,omitempty"`  // JSON schema
	DefaultModel string   `gorm:"size:256" json:"default_model,omitempty"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	SourcePath  string    `gorm:"size:512" json:"source_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Skill) TableName() string {
	return "skills"
}

// SkillPrompt stores prompt files bundled with a skill.
type SkillPrompt struct {
	Id      uint   `gorm:"primaryKey" json:"id"`
	SkillId uint   `gorm:"index" json:"skill_id"`
	Name    string `gorm:"size:256" json:"name"`
	Content string `gorm:"type:text" json:"content"`
}

func (SkillPrompt) TableName() string {
	return "skill_prompts"
}

// SkillResource stores resource files bundled with a skill.
type SkillResource struct {
	Id          uint   `gorm:"primaryKey" json:"id"`
	SkillId     uint   `gorm:"index" json:"skill_id"`
	Path        string `gorm:"size:512" json:"path"`
	ContentType string `gorm:"size:128" json:"content_type"`
	Content     []byte `gorm:"type:blob" json:"content,omitempty"`
}

func (SkillResource) TableName() string {
	return "skill_resources"
}
