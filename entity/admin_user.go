package entity

import "time"

// AdminUser Admin user
type AdminUser struct {
	Id           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:128;uniqueIndex" json:"username"` // Username
	PasswordHash string    `gorm:"size:256" json:"-"`                    // bcrypt password hash
	IsAdmin      bool      `gorm:"default:true" json:"is_admin"`         // Whether user is admin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}
