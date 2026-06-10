package model

import (
	"github.com/chuccp/go-ai-agent/entity"
	"github.com/chuccp/go-web-frame/core"
	"github.com/chuccp/go-web-frame/db"
)

type AdminUserModel struct {
	core.IModel
	db *db.DB
}

func (m *AdminUserModel) Init(d *db.DB, ctx *core.Context) error {
	m.db = d
	return m.CreateTable()
}

func (m *AdminUserModel) t() *db.Table {
	return m.db.Table((&entity.AdminUser{}).TableName())
}

// Create inserts a new admin user record.
func (m *AdminUserModel) Create(user *entity.AdminUser) error {
	return m.t().Create(user)
}

// FindByUsername looks up an admin user by username.
func (m *AdminUserModel) FindByUsername(username string) (*entity.AdminUser, error) {
	var user entity.AdminUser
	err := m.t().Where("username = ?", username).First(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// HasAdminUser checks if any admin user exists in the database.
func (m *AdminUserModel) HasAdminUser() (bool, error) {
	var count int64
	err := m.t().Where("is_admin = ?", true).Count(&count)
	return count > 0, err
}

// UpdatePassword updates the password hash for an admin user.
func (m *AdminUserModel) UpdatePassword(id uint, passwordHash string) error {
	return m.t().Where("id = ?", id).UpdateColumn("password_hash", passwordHash)
}

func (m *AdminUserModel) IsExist() (bool, error) { return false, nil }

func (m *AdminUserModel) CreateTable() error {
	return m.db.Table((&entity.AdminUser{}).TableName()).AutoMigrate(&entity.AdminUser{})
}

func (m *AdminUserModel) DeleteTable() error {
	return m.db.Migrator().DropTable(&entity.AdminUser{})
}

func (m *AdminUserModel) GetTableName() string {
	return (&entity.AdminUser{}).TableName()
}

func (m *AdminUserModel) ReNew(d *db.DB, c *core.Context) core.IModel {
	return &AdminUserModel{db: d}
}
