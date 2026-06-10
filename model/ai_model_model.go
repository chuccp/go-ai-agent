package model

import (
	"github.com/chuccp/go-ai-agent/entity"
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

func (m *AIModelModel) Create(model *entity.AIModel) error {
	return m.EntryModel.Save(model)
}

func (m *AIModelModel) FindById(id uint) (*entity.AIModel, error) {
	return m.FindByPK(id)
}

func (m *AIModelModel) List() ([]*entity.AIModel, error) {
	return m.EntryModel.Query().
		Order("category ASC, provider ASC, name ASC").
		All()
}

func (m *AIModelModel) ListByCategory(category string) ([]*entity.AIModel, error) {
	return m.EntryModel.Query().
		Where("category = ?", category).
		Order("provider ASC, name ASC").
		All()
}

func (m *AIModelModel) FindDefault(category string) (*entity.AIModel, error) {
	return m.EntryModel.Query().
		Where("category = ? AND is_default = ?", category, true).
		One()
}

func (m *AIModelModel) FindBase() ([]*entity.AIModel, error) {
	return m.EntryModel.Query().
		Where("is_base = ?", true).
		Order("id ASC").
		All()
}

func (m *AIModelModel) Update(model *entity.AIModel) error {
	return m.EntryModel.Save(model)
}

func (m *AIModelModel) Delete(id uint) error {
	return m.EntryModel.DeleteByPK(id)
}

func (m *AIModelModel) ClearDefaultByCategory(category string) error {
	return m.EntryModel.Update().
		Where("category = ? AND is_default = ?", category, true).
		UpdateColumn("is_default", false)
}

func (m *AIModelModel) ClearBase() error {
	return m.EntryModel.Update().
		Where("is_base = ?", true).
		UpdateColumn("is_base", false)
}

func (m *AIModelModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *AIModelModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *AIModelModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *AIModelModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *AIModelModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.AIModel{}).TableName()
	return &AIModelModel{
		EntryModel: fwModel.NewEntryModel[*entity.AIModel, uint](d, tableName),
	}
}
