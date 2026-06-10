package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// ==================== FlowModel ====================

type FlowModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowDefinition, uint]
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

// ==================== FlowNodeModel ====================

type FlowNodeModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowNode, uint]
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

// ==================== FlowEdgeModel ====================

type FlowEdgeModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowEdge, uint]
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

// ==================== FlowExecutionModel ====================

type FlowExecutionModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.FlowExecution, uint]
}

func (m *FlowExecutionModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.FlowExecution{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.FlowExecution, uint](d, tableName)
	return m.CreateTable()
}

func (m *FlowExecutionModel) Create(exec *entity.FlowExecution) error {
	return m.EntryModel.Save(exec)
}

func (m *FlowExecutionModel) FindById(id uint) (*entity.FlowExecution, error) {
	return m.FindByPK(id)
}

func (m *FlowExecutionModel) FindBySessionId(sessionId uint) ([]*entity.FlowExecution, error) {
	return m.EntryModel.Query().
		Where("session_id = ?", sessionId).
		Order("created_at desc").
		All()
}

func (m *FlowExecutionModel) FindByFlowId(flowId uint) ([]*entity.FlowExecution, error) {
	return m.EntryModel.Query().
		Where("flow_id = ?", flowId).
		Order("created_at desc").
		All()
}

func (m *FlowExecutionModel) Update(exec *entity.FlowExecution) error {
	return m.EntryModel.Save(exec)
}

func (m *FlowExecutionModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *FlowExecutionModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *FlowExecutionModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *FlowExecutionModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *FlowExecutionModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.FlowExecution{}).TableName()
	return &FlowExecutionModel{
		EntryModel: fwModel.NewEntryModel[*entity.FlowExecution, uint](d, tableName),
	}
}
