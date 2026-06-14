package model

import (
	"context"

	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ==================== FlowEdgeModel ====================

type FlowEdgeModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowEdge, uint]
}

func (m *FlowEdgeModel) WithContext(ctx context.Context) *FlowEdgeModel {
	return &FlowEdgeModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *FlowEdgeModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.FlowEdge{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.FlowEdge, uint](d, tableName)
	return m.CreateTable()
}

func (m *FlowEdgeModel) Create(edge *entity.FlowEdge) error {
	return m.EntryModel.Save(edge)
}

func (m *FlowEdgeModel) CreateBatch(edges []*entity.FlowEdge) error {
	return m.EntryModel.Saves(edges)
}

func (m *FlowEdgeModel) FindByFlowId(flowId uint) ([]*entity.FlowEdge, error) {
	return m.EntryModel.Query().
		Where("flow_id = ?", flowId).
		Order("id asc").
		All()
}

func (m *FlowEdgeModel) DeleteByFlowId(flowId uint) error {
	return m.EntryModel.Delete().
		Where("flow_id = ?", flowId).
		Delete()
}

func (m *FlowEdgeModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *FlowEdgeModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *FlowEdgeModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *FlowEdgeModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *FlowEdgeModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.FlowEdge{}).TableName()
	return &FlowEdgeModel{
		EntryModel: fwModel.NewEntryModel[*entity.FlowEdge, uint](d, tableName),
	}
}
