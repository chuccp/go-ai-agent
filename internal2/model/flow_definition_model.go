package model

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal2/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ==================== FlowModel ====================

type FlowModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowDefinition, uint]
}

func (m *FlowModel) WithContext(ctx context.Context) *FlowModel {
	return &FlowModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *FlowModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.FlowDefinition{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.FlowDefinition, uint](d, tableName)
	return m.CreateTable()
}

func (m *FlowModel) Create(flow *entity.FlowDefinition) error {
	return m.EntryModel.Save(flow)
}

func (m *FlowModel) FindById(id uint) (*entity.FlowDefinition, error) {
	return m.FindByPK(id)
}

func (m *FlowModel) List() ([]*entity.FlowDefinition, error) {
	return m.EntryModel.Query().
		Order("updated_at desc").
		All()
}

func (m *FlowModel) ListByCategory(category string) ([]*entity.FlowDefinition, error) {
	return m.EntryModel.Query().
		Where("category = ?", category).
		Order("updated_at desc").
		All()
}

func (m *FlowModel) Update(flow *entity.FlowDefinition) error {
	return m.EntryModel.Save(flow)
}

func (m *FlowModel) Delete(id uint) error {
	return m.EntryModel.DeleteByPK(id)
}

func (m *FlowModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *FlowModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *FlowModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *FlowModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *FlowModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.FlowDefinition{}).TableName()
	return &FlowModel{
		EntryModel: fwModel.NewEntryModel[*entity.FlowDefinition, uint](d, tableName),
	}
}
