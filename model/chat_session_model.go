package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
)

// ChatSessionModel 聊天会话 Model
type ChatSessionModel struct {
	core.IModel
	db *db.DB
}

func (m *ChatSessionModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.ChatSession{}).TableName()).AutoMigrate(&entity.ChatSession{})
}

func (m *ChatSessionModel) t() *db.Table {
	return m.db.Table((&entity.ChatSession{}).TableName())
}

func (m *ChatSessionModel) Create(session *entity.ChatSession) error {
	return m.t().Create(session)
}

func (m *ChatSessionModel) FindById(id uint) (*entity.ChatSession, error) {
	var session entity.ChatSession
	err := m.t().First(&session, id)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (m *ChatSessionModel) List() ([]*entity.ChatSession, error) {
	var sessions []*entity.ChatSession
	err := m.t().Order("updated_at desc").Find(&sessions)
	return sessions, err
}

func (m *ChatSessionModel) Delete(id uint) error {
	m.db.Table((&entity.ChatMessage{}).TableName()).Where("session_id = ?", id).Delete(&entity.ChatMessage{})
	return m.t().Delete(&entity.ChatSession{}, id)
}

func (m *ChatSessionModel) Update(session *entity.ChatSession) error {
	return m.t().Save(session)
}

func (m *ChatSessionModel) IsExist() (bool, error) {
	return false, nil
}

func (m *ChatSessionModel) CreateTable() error {
	return m.db.Table((&entity.ChatSession{}).TableName()).AutoMigrate(&entity.ChatSession{})
}

func (m *ChatSessionModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.ChatSession{})
}

func (m *ChatSessionModel) GetTableName() string {
	return (&entity.ChatSession{}).TableName()
}

func (m *ChatSessionModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &ChatSessionModel{db: d}
}

// ChatMessageModel 聊天消息 Model
type ChatMessageModel struct {
	core.IModel
	db *db.DB
}

func (m *ChatMessageModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.ChatMessage{}).TableName()).AutoMigrate(&entity.ChatMessage{})
}

func (m *ChatMessageModel) t() *db.Table {
	return m.db.Table((&entity.ChatMessage{}).TableName())
}

func (m *ChatMessageModel) Create(msg *entity.ChatMessage) error {
	return m.t().Create(msg)
}

func (m *ChatMessageModel) CreateBatch(msgs []*entity.ChatMessage) error {
	for _, msg := range msgs {
		if err := m.t().Create(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *ChatMessageModel) FindBySessionId(sessionId uint) ([]*entity.ChatMessage, error) {
	var messages []*entity.ChatMessage
	err := m.t().Where("session_id = ?", sessionId).Order("created_at asc").Find(&messages)
	return messages, err
}

func (m *ChatMessageModel) DeleteBySessionId(sessionId uint) error {
	return m.t().Where("session_id = ?", sessionId).Delete(&entity.ChatMessage{})
}

func (m *ChatMessageModel) IsExist() (bool, error) {
	return false, nil
}

func (m *ChatMessageModel) CreateTable() error {
	return m.db.Table((&entity.ChatMessage{}).TableName()).AutoMigrate(&entity.ChatMessage{})
}

func (m *ChatMessageModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.ChatMessage{})
}

func (m *ChatMessageModel) GetTableName() string {
	return (&entity.ChatMessage{}).TableName()
}

func (m *ChatMessageModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &ChatMessageModel{db: d}
}
