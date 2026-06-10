package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
)

// ==================== FlowModel ====================

type FlowModel struct {
	core.IModel
	db *db.DB
}

func (m *FlowModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.FlowDefinition{}).TableName()).AutoMigrate(&entity.FlowDefinition{})
}

func (m *FlowModel) t() *db.Table {
	return m.db.Table((&entity.FlowDefinition{}).TableName())
}

func (m *FlowModel) Create(flow *entity.FlowDefinition) error {
	return m.t().Create(flow)
}

func (m *FlowModel) FindById(id uint) (*entity.FlowDefinition, error) {
	var flow entity.FlowDefinition
	err := m.t().First(&flow, id)
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

func (m *FlowModel) List() ([]*entity.FlowDefinition, error) {
	var flows []*entity.FlowDefinition
	err := m.t().Order("updated_at desc").Find(&flows)
	return flows, err
}

func (m *FlowModel) ListByCategory(category string) ([]*entity.FlowDefinition, error) {
	var flows []*entity.FlowDefinition
	err := m.t().Where("category = ?", category).Order("updated_at desc").Find(&flows)
	return flows, err
}

func (m *FlowModel) Update(flow *entity.FlowDefinition) error {
	return m.t().Save(flow)
}

func (m *FlowModel) Delete(id uint) error {
	return m.t().Delete(&entity.FlowDefinition{}, id)
}

func (m *FlowModel) IsExist() (bool, error) {
	return false, nil
}

func (m *FlowModel) CreateTable() error {
	return m.db.Table((&entity.FlowDefinition{}).TableName()).AutoMigrate(&entity.FlowDefinition{})
}

func (m *FlowModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.FlowDefinition{})
}

func (m *FlowModel) GetTableName() string {
	return (&entity.FlowDefinition{}).TableName()
}

func (m *FlowModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &FlowModel{db: d}
}

// ==================== FlowNodeModel ====================

type FlowNodeModel struct {
	core.IModel
	db *db.DB
}

func (m *FlowNodeModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.FlowNode{}).TableName()).AutoMigrate(&entity.FlowNode{})
}

func (m *FlowNodeModel) t() *db.Table {
	return m.db.Table((&entity.FlowNode{}).TableName())
}

func (m *FlowNodeModel) CreateBatch(nodes []*entity.FlowNode) error {
	for _, n := range nodes {
		if err := m.t().Create(n); err != nil {
			return err
		}
	}
	return nil
}

func (m *FlowNodeModel) Create(node *entity.FlowNode) error {
	return m.t().Create(node)
}

func (m *FlowNodeModel) FindByFlowId(flowId uint) ([]*entity.FlowNode, error) {
	var nodes []*entity.FlowNode
	err := m.t().Where("flow_id = ?", flowId).Order("id asc").Find(&nodes)
	return nodes, err
}

func (m *FlowNodeModel) DeleteByFlowId(flowId uint) error {
	return m.t().Where("flow_id = ?", flowId).Delete(&entity.FlowNode{})
}

func (m *FlowNodeModel) Update(node *entity.FlowNode) error {
	return m.t().Save(node)
}

func (m *FlowNodeModel) IsExist() (bool, error) {
	return false, nil
}

func (m *FlowNodeModel) CreateTable() error {
	return m.db.Table((&entity.FlowNode{}).TableName()).AutoMigrate(&entity.FlowNode{})
}

func (m *FlowNodeModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.FlowNode{})
}

func (m *FlowNodeModel) GetTableName() string {
	return (&entity.FlowNode{}).TableName()
}

func (m *FlowNodeModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &FlowNodeModel{db: d}
}

// ==================== FlowEdgeModel ====================

type FlowEdgeModel struct {
	core.IModel
	db *db.DB
}

func (m *FlowEdgeModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.FlowEdge{}).TableName()).AutoMigrate(&entity.FlowEdge{})
}

func (m *FlowEdgeModel) t() *db.Table {
	return m.db.Table((&entity.FlowEdge{}).TableName())
}

func (m *FlowEdgeModel) CreateBatch(edges []*entity.FlowEdge) error {
	for _, e := range edges {
		if err := m.t().Create(e); err != nil {
			return err
		}
	}
	return nil
}

func (m *FlowEdgeModel) Create(edge *entity.FlowEdge) error {
	return m.t().Create(edge)
}

func (m *FlowEdgeModel) FindByFlowId(flowId uint) ([]*entity.FlowEdge, error) {
	var edges []*entity.FlowEdge
	err := m.t().Where("flow_id = ?", flowId).Order("id asc").Find(&edges)
	return edges, err
}

func (m *FlowEdgeModel) DeleteByFlowId(flowId uint) error {
	return m.t().Where("flow_id = ?", flowId).Delete(&entity.FlowEdge{})
}

func (m *FlowEdgeModel) IsExist() (bool, error) {
	return false, nil
}

func (m *FlowEdgeModel) CreateTable() error {
	return m.db.Table((&entity.FlowEdge{}).TableName()).AutoMigrate(&entity.FlowEdge{})
}

func (m *FlowEdgeModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.FlowEdge{})
}

func (m *FlowEdgeModel) GetTableName() string {
	return (&entity.FlowEdge{}).TableName()
}

func (m *FlowEdgeModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &FlowEdgeModel{db: d}
}

// ==================== FlowExecutionModel ====================

type FlowExecutionModel struct {
	core.IModel
	db *db.DB
}

func (m *FlowExecutionModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.db.Table((&entity.FlowExecution{}).TableName()).AutoMigrate(&entity.FlowExecution{})
}

func (m *FlowExecutionModel) t() *db.Table {
	return m.db.Table((&entity.FlowExecution{}).TableName())
}

func (m *FlowExecutionModel) Create(exec *entity.FlowExecution) error {
	return m.t().Create(exec)
}

func (m *FlowExecutionModel) FindById(id uint) (*entity.FlowExecution, error) {
	var exec entity.FlowExecution
	err := m.t().First(&exec, id)
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (m *FlowExecutionModel) FindBySessionId(sessionId uint) ([]*entity.FlowExecution, error) {
	var execs []*entity.FlowExecution
	err := m.t().Where("session_id = ?", sessionId).Order("created_at desc").Find(&execs)
	return execs, err
}

func (m *FlowExecutionModel) FindByFlowId(flowId uint) ([]*entity.FlowExecution, error) {
	var execs []*entity.FlowExecution
	err := m.t().Where("flow_id = ?", flowId).Order("created_at desc").Find(&execs)
	return execs, err
}

func (m *FlowExecutionModel) Update(exec *entity.FlowExecution) error {
	return m.t().Save(exec)
}

func (m *FlowExecutionModel) IsExist() (bool, error) {
	return false, nil
}

func (m *FlowExecutionModel) CreateTable() error {
	return m.db.Table((&entity.FlowExecution{}).TableName()).AutoMigrate(&entity.FlowExecution{})
}

func (m *FlowExecutionModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.FlowExecution{})
}

func (m *FlowExecutionModel) GetTableName() string {
	return (&entity.FlowExecution{}).TableName()
}

func (m *FlowExecutionModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &FlowExecutionModel{db: d}
}
