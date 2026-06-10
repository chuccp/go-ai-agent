package entity

import "time"

// ChatSession 聊天会话
type ChatSession struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:256" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	SessionId uint      `gorm:"index" json:"session_id"`
	Role      string    `gorm:"size:32" json:"role"`
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
