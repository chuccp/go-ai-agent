package model

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// SkillModel manages Skill entities.
type SkillModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.Skill, uint]
}

func (m *SkillModel) WithContext(ctx context.Context) *SkillModel {
	if m.EntryModel == nil {
		return &SkillModel{IModel: m.IModel}
	}
	return &SkillModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *SkillModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.Skill{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.Skill, uint](d, tableName)
	return m.CreateTable()
}

func (m *SkillModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *SkillModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *SkillModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *SkillModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.Skill{}).TableName()
	return &SkillModel{
		EntryModel: fwModel.NewEntryModel[*entity.Skill, uint](d, tableName),
	}
}

// SkillPromptModel manages SkillPrompt entities.
type SkillPromptModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.SkillPrompt, uint]
}

func (m *SkillPromptModel) WithContext(ctx context.Context) *SkillPromptModel {
	if m.EntryModel == nil {
		return &SkillPromptModel{IModel: m.IModel}
	}
	return &SkillPromptModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *SkillPromptModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.SkillPrompt{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.SkillPrompt, uint](d, tableName)
	return m.CreateTable()
}

func (m *SkillPromptModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *SkillPromptModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *SkillPromptModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *SkillPromptModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.SkillPrompt{}).TableName()
	return &SkillPromptModel{
		EntryModel: fwModel.NewEntryModel[*entity.SkillPrompt, uint](d, tableName),
	}
}

// SkillResourceModel manages SkillResource entities.
type SkillResourceModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.SkillResource, uint]
}

func (m *SkillResourceModel) WithContext(ctx context.Context) *SkillResourceModel {
	if m.EntryModel == nil {
		return &SkillResourceModel{IModel: m.IModel}
	}
	return &SkillResourceModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *SkillResourceModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.SkillResource{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.SkillResource, uint](d, tableName)
	return m.CreateTable()
}

func (m *SkillResourceModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *SkillResourceModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *SkillResourceModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *SkillResourceModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.SkillResource{}).TableName()
	return &SkillResourceModel{
		EntryModel: fwModel.NewEntryModel[*entity.SkillResource, uint](d, tableName),
	}
}
