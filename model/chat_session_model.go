package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ChatSessionModel 聊天会话 Model
type ChatSessionModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.ChatSession, uint]
	messageModel *ChatMessageModel
}

func (m *ChatSessionModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.ChatSession{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.ChatSession, uint](d, tableName)
	m.messageModel = core.GetModel[*ChatMessageModel](ctx)
	return m.CreateTable()
}

func (m *ChatSessionModel) Create(session *entity.ChatSession) error {
	return m.EntryModel.Save(session)
}

func (m *ChatSessionModel) FindById(id uint) (*entity.ChatSession, error) {
	return m.FindByPK(id)
}

func (m *ChatSessionModel) List() ([]*entity.ChatSession, error) {
	return m.EntryModel.Query().
		Order("updated_at desc").
		All()
}

func (m *ChatSessionModel) Delete(id uint) error {
	if err := m.messageModel.DeleteBySessionId(id); err != nil {
		return err
	}
	return m.EntryModel.DeleteByPK(id)
}

func (m *ChatSessionModel) Update(session *entity.ChatSession) error {
	return m.EntryModel.Save(session)
}

func (m *ChatSessionModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *ChatSessionModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *ChatSessionModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *ChatSessionModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *ChatSessionModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.ChatSession{}).TableName()
	return &ChatSessionModel{
		EntryModel: fwModel.NewEntryModel[*entity.ChatSession, uint](d, tableName),
	}
}

// ChatMessageModel 聊天消息 Model
type ChatMessageModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.ChatMessage, uint]
}

func (m *ChatMessageModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.ChatMessage{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.ChatMessage, uint](d, tableName)
	return m.CreateTable()
}

func (m *ChatMessageModel) Create(msg *entity.ChatMessage) error {
	return m.EntryModel.Save(msg)
}

func (m *ChatMessageModel) CreateBatch(msgs []*entity.ChatMessage) error {
	return m.EntryModel.Saves(msgs)
}

func (m *ChatMessageModel) FindBySessionId(sessionId uint) ([]*entity.ChatMessage, error) {
	return m.EntryModel.Query().
		Where("session_id = ?", sessionId).
		Order("created_at asc").
		All()
}

func (m *ChatMessageModel) DeleteBySessionId(sessionId uint) error {
	return m.EntryModel.Delete().
		Where("session_id = ?", sessionId).
		Delete()
}

func (m *ChatMessageModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *ChatMessageModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *ChatMessageModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *ChatMessageModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *ChatMessageModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.ChatMessage{}).TableName()
	return &ChatMessageModel{
		EntryModel: fwModel.NewEntryModel[*entity.ChatMessage, uint](d, tableName),
	}
}
