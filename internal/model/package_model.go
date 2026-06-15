package model

import (
	"context"

	"github.com/chuccp/go-ai-agent/internal/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

// PackageModel manages Package entities.
type PackageModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.Package, uint]
}

func (m *PackageModel) WithContext(ctx context.Context) *PackageModel {
	if m.EntryModel == nil {
		return &PackageModel{IModel: m.IModel}
	}
	return &PackageModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *PackageModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.Package{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.Package, uint](d, tableName)
	return m.CreateTable()
}

func (m *PackageModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *PackageModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *PackageModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *PackageModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.Package{}).TableName()
	return &PackageModel{
		EntryModel: fwModel.NewEntryModel[*entity.Package, uint](d, tableName),
	}
}

// PackageResourceModel manages PackageResource entities.
type PackageResourceModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.PackageResource, uint]
}

func (m *PackageResourceModel) WithContext(ctx context.Context) *PackageResourceModel {
	if m.EntryModel == nil {
		return &PackageResourceModel{IModel: m.IModel}
	}
	return &PackageResourceModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *PackageResourceModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.PackageResource{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.PackageResource, uint](d, tableName)
	return m.CreateTable()
}

func (m *PackageResourceModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *PackageResourceModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *PackageResourceModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *PackageResourceModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.PackageResource{}).TableName()
	return &PackageResourceModel{
		EntryModel: fwModel.NewEntryModel[*entity.PackageResource, uint](d, tableName),
	}
}

// PackageConfigModel manages PackageConfig entities.
type PackageConfigModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.PackageConfig, uint]
}

func (m *PackageConfigModel) WithContext(ctx context.Context) *PackageConfigModel {
	if m.EntryModel == nil {
		return &PackageConfigModel{IModel: m.IModel}
	}
	return &PackageConfigModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

func (m *PackageConfigModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.PackageConfig{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.PackageConfig, uint](d, tableName)
	return m.CreateTable()
}

func (m *PackageConfigModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *PackageConfigModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *PackageConfigModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *PackageConfigModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.PackageConfig{}).TableName()
	return &PackageConfigModel{
		EntryModel: fwModel.NewEntryModel[*entity.PackageConfig, uint](d, tableName),
	}
}
