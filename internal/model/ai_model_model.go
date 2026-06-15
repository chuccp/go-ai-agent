package model

import (
	"context"
	"errors"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

type AIModelModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.AIModel, uint]
}

func (m *AIModelModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.AIModel{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.AIModel, uint](d, tableName)
	return m.CreateTable()
}

func (m *AIModelModel) WithContext(ctx context.Context) *AIModelModel {
	if m.EntryModel == nil {
		return &AIModelModel{IModel: m.IModel}
	}
	return &AIModelModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

var errNotInitialized = errors.New("AI model not initialized: database not configured")

// entry returns the underlying EntryModel, or an error if the model has not been
// initialized (e.g. during first-run before the database is configured).
func (m *AIModelModel) entry() (*fwModel.EntryModel[*entity.AIModel, uint], error) {
	if m.EntryModel == nil {
		return nil, errNotInitialized
	}
	return m.EntryModel, nil
}

func (m *AIModelModel) Create(model *entity.AIModel) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(model)
}

func (m *AIModelModel) FindById(id uint) (*entity.AIModel, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.FindByPK(id)
}

func (m *AIModelModel) List() ([]*entity.AIModel, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Order("category ASC, provider ASC, name ASC").
		All()
}

func (m *AIModelModel) ListByCategory(category string) ([]*entity.AIModel, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Where("category = ?", category).
		Order("provider ASC, name ASC").
		All()
}

func (m *AIModelModel) FindDefault(category string) (*entity.AIModel, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Where("category = ? AND is_default = ?", category, true).
		One()
}

func (m *AIModelModel) FindBase() ([]*entity.AIModel, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.Query().
		Where("is_base = ?", true).
		Order("id ASC").
		All()
}

func (m *AIModelModel) Update(model *entity.AIModel) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(model)
}

func (m *AIModelModel) Delete(id uint) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.DeleteByPK(id)
}

func (m *AIModelModel) ClearDefaultByCategory(category string) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Update().
		Where("category = ? AND is_default = ?", category, true).
		UpdateColumn("is_default", false)
}

func (m *AIModelModel) ClearBase() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Update().
		Where("is_base = ?", true).
		UpdateColumn("is_base", false)
}

func (m *AIModelModel) IsExist() (bool, error) {
	em, err := m.entry()
	if err != nil {
		return false, err
	}
	return em.IsExist()
}

func (m *AIModelModel) CreateTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.CreateTable()
}

func (m *AIModelModel) DeleteTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.DeleteTable()
}

func (m *AIModelModel) GetTableName() string {
	if m.EntryModel == nil {
		return ""
	}
	return m.EntryModel.GetTableName()
}

func (m *AIModelModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.AIModel{}).TableName()
	return &AIModelModel{
		EntryModel: fwModel.NewEntryModel[*entity.AIModel, uint](d, tableName),
	}
}
