package model

import (
	"context"
	"errors"

	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
	fwModel "github.com/chuccp/go-web-frame/model"
)

type AdminUserModel struct {
	core.IModel
	*fwModel.EntryModel[*entity.AdminUser, uint]
}

func (m *AdminUserModel) Init(d *db.DB, ctx *core.Context) error {
	tableName := (&entity.AdminUser{}).TableName()
	m.EntryModel = fwModel.NewEntryModel[*entity.AdminUser, uint](d, tableName)
	return m.CreateTable()
}

func (m *AdminUserModel) WithContext(ctx context.Context) *AdminUserModel {
	if m.EntryModel == nil {
		return &AdminUserModel{IModel: m.IModel}
	}
	return &AdminUserModel{
		IModel:     m.IModel,
		EntryModel: m.EntryModel.WithContext(ctx),
	}
}

var errAdminNotInitialized = errors.New("admin user model not initialized: database not configured")

func (m *AdminUserModel) entry() (*fwModel.EntryModel[*entity.AdminUser, uint], error) {
	if m.EntryModel == nil {
		return nil, errAdminNotInitialized
	}
	return m.EntryModel, nil
}

func (m *AdminUserModel) Create(user *entity.AdminUser) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.Save(user)
}

func (m *AdminUserModel) FindByUsername(username string) (*entity.AdminUser, error) {
	em, err := m.entry()
	if err != nil {
		return nil, err
	}
	return em.FindOne("username = ?", username)
}

func (m *AdminUserModel) HasAdminUser() (bool, error) {
	em, err := m.entry()
	if err != nil {
		return false, err
	}
	count, err := em.Query().Where("is_admin = ?", true).Count()
	return count > 0, err
}

func (m *AdminUserModel) UpdatePassword(id uint, passwordHash string) error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.UpdateColumn(id, "password_hash", passwordHash)
}

func (m *AdminUserModel) IsExist() (bool, error) {
	em, err := m.entry()
	if err != nil {
		return false, err
	}
	return em.IsExist()
}

func (m *AdminUserModel) CreateTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.CreateTable()
}

func (m *AdminUserModel) DeleteTable() error {
	em, err := m.entry()
	if err != nil {
		return err
	}
	return em.DeleteTable()
}

func (m *AdminUserModel) GetTableName() string {
	if m.EntryModel == nil {
		return ""
	}
	return m.EntryModel.GetTableName()
}

func (m *AdminUserModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.AdminUser{}).TableName()
	return &AdminUserModel{
		EntryModel: fwModel.NewEntryModel[*entity.AdminUser, uint](d, tableName),
	}
}
