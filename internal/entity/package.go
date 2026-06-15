package entity

import "time"

// Package is a flow-centric container that can hold flows, skills, resources and config.
type Package struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	PackageId   string    `gorm:"size:128;uniqueIndex" json:"package_id"` // identifier from meta.json
	Name        string    `gorm:"size:256" json:"name"`
	Version     string    `gorm:"size:64" json:"version"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"size:512" json:"icon,omitempty"`
	Kind        string    `gorm:"size:64" json:"kind"`
	Meta        string    `gorm:"type:text" json:"meta,omitempty"`
	SourcePath  string    `gorm:"size:512" json:"source_path,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Package) TableName() string {
	return "packages"
}

// PackageResource stores arbitrary resources bundled with a package.
type PackageResource struct {
	Id          uint      `gorm:"primaryKey" json:"id"`
	PackageId   uint      `gorm:"index" json:"package_id"`
	Path        string    `gorm:"size:512" json:"path"`
	ContentType string    `gorm:"size:128" json:"content_type"`
	Content     []byte    `gorm:"type:blob" json:"content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (PackageResource) TableName() string {
	return "package_resources"
}

// PackageConfig stores runtime configuration bundled with a package.
type PackageConfig struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	PackageId uint      `gorm:"index" json:"package_id"`
	Key       string    `gorm:"size:256" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	IsSecret  bool      `gorm:"default:false" json:"is_secret"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PackageConfig) TableName() string {
	return "package_configs"
}
