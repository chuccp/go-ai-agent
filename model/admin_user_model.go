package model

import (
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

func (m *AdminUserModel) Create(user *entity.AdminUser) error {
	return m.EntryModel.Save(user)
}

func (m *AdminUserModel) FindByUsername(username string) (*entity.AdminUser, error) {
	return m.EntryModel.FindOne("username = ?", username)
}

func (m *AdminUserModel) HasAdminUser() (bool, error) {
	count, err := m.EntryModel.Query().Where("is_admin = ?", true).Count()
	return count > 0, err
}

func (m *AdminUserModel) UpdatePassword(id uint, passwordHash string) error {
	return m.EntryModel.UpdateColumn(id, "password_hash", passwordHash)
}

func (m *AdminUserModel) IsExist() (bool, error) {
	return m.EntryModel.IsExist()
}

func (m *AdminUserModel) CreateTable() error {
	return m.EntryModel.CreateTable()
}

func (m *AdminUserModel) DeleteTable() error {
	return m.EntryModel.DeleteTable()
}

func (m *AdminUserModel) GetTableName() string {
	return m.EntryModel.GetTableName()
}

func (m *AdminUserModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	tableName := (&entity.AdminUser{}).TableName()
	return &AdminUserModel{
		EntryModel: fwModel.NewEntryModel[*entity.AdminUser, uint](d, tableName),
	}
}
