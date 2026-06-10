package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
)

type AIModelModel struct {
	core.IModel
	db *db.DB
}

func (m *AIModelModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.AIModel{}).TableName()).AutoMigrate(&entity.AIModel{})
}

func (m *AIModelModel) t() *db.Table {
	return m.db.Table((&entity.AIModel{}).TableName())
}

func (m *AIModelModel) Create(model *entity.AIModel) error {
	return m.t().Create(model)
}

func (m *AIModelModel) FindById(id uint) (*entity.AIModel, error) {
	var model entity.AIModel
	err := m.t().First(&model, id)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (m *AIModelModel) List() ([]*entity.AIModel, error) {
	var models []*entity.AIModel
	err := m.t().Order("category ASC, provider ASC, name ASC").Find(&models)
	return models, err
}

func (m *AIModelModel) ListByCategory(category string) ([]*entity.AIModel, error) {
	var models []*entity.AIModel
	err := m.t().Where("category = ?", category).Order("provider ASC, name ASC").Find(&models)
	return models, err
}

func (m *AIModelModel) FindDefault(category string) (*entity.AIModel, error) {
	var model entity.AIModel
	err := m.t().Where("category = ? AND is_default = ?", category, true).First(&model)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (m *AIModelModel) FindBase() ([]*entity.AIModel, error) {
	var models []*entity.AIModel
	err := m.t().Where("is_base = ?", true).Order("id ASC").Find(&models)
	return models, err
}

func (m *AIModelModel) Update(model *entity.AIModel) error {
	return m.t().Save(model)
}

func (m *AIModelModel) Delete(id uint) error {
	return m.t().Delete(&entity.AIModel{}, id)
}

func (m *AIModelModel) ClearDefaultByCategory(category string) error {
	return m.t().Where("category = ? AND is_default = ?", category, true).UpdateColumn("is_default", false)
}

func (m *AIModelModel) ClearBase() error {
	return m.t().Where("is_base = ?", true).UpdateColumn("is_base", false)
}

func (m *AIModelModel) IsExist() (bool, error) { return false, nil }

func (m *AIModelModel) CreateTable() error {
	return m.db.Table((&entity.AIModel{}).TableName()).AutoMigrate(&entity.AIModel{})
}

func (m *AIModelModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.AIModel{})
}

func (m *AIModelModel) GetTableName() string {
	return (&entity.AIModel{}).TableName()
}

func (m *AIModelModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &AIModelModel{db: d}
}
