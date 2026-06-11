package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

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
