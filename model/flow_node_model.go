package model

import (
	"context"

	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ==================== FlowNodeModel ====================

type FlowNodeModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowNode, uint]
}

func (m *FlowNodeModel) WithContext(ctx context.Context) *FlowNodeModel {
	return &FlowNodeModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *FlowNodeModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.FlowNode{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.FlowNode, uint](d, tableName)
	return m.CreateTable()
}

func (m *FlowNodeModel) Create(node *entity.FlowNode) error {
	return m.EntryModel.Save(node)
}

func (m *FlowNodeModel) CreateBatch(nodes []*entity.FlowNode) error {
	return m.EntryModel.Saves(nodes)
}

func (m *FlowNodeModel) FindByFlowId(flowId uint) ([]*entity.FlowNode, error) {
	return m.EntryModel.Query().
		Where("flow_id = ?", flowId).
		Order("id asc").
		All()
}

func (m *FlowNodeModel) DeleteByFlowId(flowId uint) error {
	return m.EntryModel.Delete().
		Where("flow_id = ?", flowId).
		Delete()
}

func (m *FlowNodeModel) Update(node *entity.FlowNode) error {
	return m.EntryModel.Save(node)
}

func (m *FlowNodeModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *FlowNodeModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *FlowNodeModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *FlowNodeModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *FlowNodeModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.FlowNode{}).TableName()
	return &FlowNodeModel{
		EntryModel: fwModel.NewEntryModel[*entity.FlowNode, uint](d, tableName),
	}
}
