package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 通用字段
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Department 部门（支持多级）
type Department struct {
	BaseModel
	Name     string       `gorm:"size:100;not null" json:"name"`
	ParentID *uint        `json:"parent_id"`
	Parent   *Department  `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// Role 角色
type Role struct {
	BaseModel
	Name        string       `gorm:"size:100;uniqueIndex;not null" json:"name"`
	Description string       `gorm:"size:255" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// Permission 权限
type Permission struct {
	BaseModel
	Resource string `gorm:"size:100;not null" json:"resource"` // e.g. order
	Action   string `gorm:"size:100;not null" json:"action"`   // e.g. create
	Roles    []Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

// User 用户
type User struct {
	BaseModel
	Name         string      `gorm:"size:100;not null" json:"name"`
	PasswordHash string      `gorm:"size:255;not null" json:"-"`
	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	RoleID       *uint       `json:"role_id"`
	Role         *Role       `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	IsActive     bool        `gorm:"default:true" json:"is_active"`
}
