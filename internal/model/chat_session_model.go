package model

import (
	"context"
	"errors"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ChatSessionModel chat session model
type ChatSessionModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.ChatSession, uint]
	messageModel *ChatMessageModel
}

func (m *ChatSessionModel) WithContext(ctx context.Context) *ChatSessionModel {
	var mm *ChatMessageModel
	if m.messageModel != nil {
		mm = m.messageModel.WithContext(ctx)
	}
	if m.EntryModel == nil {
		return &ChatSessionModel{IModel: m.IModel, messageModel: mm}
	}
	return &ChatSessionModel{
		IModel:       m.IModel,
		EntryModel:   m.EntryModel.WithContext(ctx),
		messageModel: mm,
	}
}

var errSessionNotInitialized = errors.New("chat session model not initialized: database not configured")

func (m *ChatSessionModel) entry() (*fwModel.EntryModel[*entity.ChatSession, uint], error) {
	if m.EntryModel == nil {
		return nil, errSessionNotInitialized
	}
	return m.EntryModel, nil
}

func (m *ChatSessionModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.ChatSession{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.ChatSession, uint](d, tableName)
	m.messageModel = core.GetModel[*ChatMessageModel](ctx)
	return m.CreateTable()
}

func (m *ChatSessionModel) Create(session *entity.ChatSession) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(session)
}

func (m *ChatSessionModel) FindById(id uint) (*entity.ChatSession, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.FindByPK(id)
}

func (m *ChatSessionModel) List() ([]*entity.ChatSession, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Order("updated_at desc").
		All()
}

func (m *ChatSessionModel) Delete(id uint) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	if m.messageModel != nil {
		if err := m.messageModel.DeleteBySessionId(id); err != nil {
			return err
		}
	}
	return em.DeleteByPK(id)
}

func (m *ChatSessionModel) Update(session *entity.ChatSession) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(session)
}

func (m *ChatSessionModel) IsExist() (bool, error) {
	em, err := m.entry()
	if err != nil {
		return false, err
	}
	return em.IsExist()
}

func (m *ChatSessionModel) CreateTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.CreateTable()
}

func (m *ChatSessionModel) DeleteTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.DeleteTable()
}

func (m *ChatSessionModel) GetTableName() string {
	if m.EntryModel == nil {
		return ""
	}
	return m.EntryModel.GetTableName()
}

func (m *ChatSessionModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.ChatSession{}).TableName()
	return &ChatSessionModel{
		EntryModel: fwModel.NewEntryModel[*entity.ChatSession, uint](d, tableName),
	}
}

// ChatMessageModel chat message model
type ChatMessageModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.ChatMessage, uint]
}

func (m *ChatMessageModel) WithContext(ctx context.Context) *ChatMessageModel {
	if m.EntryModel == nil {
		return &ChatMessageModel{IModel: m.IModel}
	}
	return &ChatMessageModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

var errMessageNotInitialized = errors.New("chat message model not initialized: database not configured")

func (m *ChatMessageModel) entry() (*fwModel.EntryModel[*entity.ChatMessage, uint], error) {
	if m.EntryModel == nil {
		return nil, errMessageNotInitialized
	}
	return m.EntryModel, nil
}

func (m *ChatMessageModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.ChatMessage{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.ChatMessage, uint](d, tableName)
	return m.CreateTable()
}

func (m *ChatMessageModel) Create(msg *entity.ChatMessage) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(msg)
}

func (m *ChatMessageModel) CreateBatch(msgs []*entity.ChatMessage) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Saves(msgs)
}

func (m *ChatMessageModel) FindBySessionId(sessionId uint) ([]*entity.ChatMessage, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Where("session_id = ?", sessionId).
		Order("created_at asc").
		All()
}

func (m *ChatMessageModel) DeleteBySessionId(sessionId uint) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Delete().
		Where("session_id = ?", sessionId).
		Delete()
}

func (m *ChatMessageModel) IsExist() (bool, error) {
	em, err := m.entry()
	if err != nil {
		return false, err
	}
	return em.IsExist()
}

func (m *ChatMessageModel) CreateTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.CreateTable()
}

func (m *ChatMessageModel) DeleteTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.DeleteTable()
}

func (m *ChatMessageModel) GetTableName() string {
	if m.EntryModel == nil {
		return ""
	}
	return m.EntryModel.GetTableName()
}

func (m *ChatMessageModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.ChatMessage{}).TableName()
	return &ChatMessageModel{
		EntryModel: fwModel.NewEntryModel[*entity.ChatMessage, uint](d, tableName),
	}
}
