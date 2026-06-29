package database

import (
	"log"

	"github.com/sui/scan-report/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// Seed 在数据库为空时写入初始数据，已有数据则跳过（幂等）
func Seed() {
	seedDepartments()
	seedPermissions()
	seedRoles()
	seedUsers()
	seedProcesses()
	log.Println("seed data applied")
}

// ─── 部门 ────────────────────────────────────────────────────────────────────

func seedDepartments() {
	var count int64
	DB.Model(&model.Department{}).Count(&count)
	if count > 0 {
		return
	}

	depts := []model.Department{
		{Name: "管理部"},
		{Name: "生产部"},
		{Name: "品质部"},
		{Name: "仓储部"},
	}
	if err := DB.Create(&depts).Error; err != nil {
		log.Printf("seed departments error: %v", err)
	}
}

// ─── 权限 ────────────────────────────────────────────────────────────────────

// 所有资源及其操作，与 RBAC 中间件的 "resource:action" 对应
var allPermissions = []struct{ Resource, Action string }{
	{"user", "read"},
	{"user", "create"},
	{"user", "update"},
	{"user", "delete"},
	{"department", "read"},
	{"department", "create"},
	{"department", "update"},
	{"department", "delete"},
	{"role", "read"},
	{"role", "create"},
	{"role", "update"},
	{"role", "delete"},
	{"process", "read"},
	{"process", "create"},
	{"process", "update"},
	{"process", "delete"},
	{"order", "read"},
	{"order", "create"},
	{"order", "update"},
	{"order", "delete"},
	{"report", "read"},
	{"report", "create"},
}

func seedPermissions() {
	var count int64
	DB.Model(&model.Permission{}).Count(&count)
	if count > 0 {
		return
	}

	for _, p := range allPermissions {
		perm := model.Permission{Resource: p.Resource, Action: p.Action}
		if err := DB.Create(&perm).Error; err != nil {
			log.Printf("seed permission %s:%s error: %v", p.Resource, p.Action, err)
		}
	}
}

// ─── 角色 ────────────────────────────────────────────────────────────────────

func seedRoles() {
	var count int64
	DB.Model(&model.Role{}).Count(&count)
	if count > 0 {
		return
	}

	// 管理员：所有权限
	var allPerms []model.Permission
	DB.Find(&allPerms)

	admin := model.Role{
		Name:        "管理员",
		Description: "拥有全部权限",
		Permissions: allPerms,
	}
	if err := DB.Create(&admin).Error; err != nil {
		log.Printf("seed role admin error: %v", err)
	}

	// 文员：工单读写 + 报工查看
	var clerkPerms []model.Permission
	DB.Where("(resource = 'order') OR (resource = 'report' AND action = 'read') OR (resource = 'process' AND action = 'read')").
		Find(&clerkPerms)

	clerk := model.Role{
		Name:        "文员",
		Description: "工单管理、查看报工进度",
		Permissions: clerkPerms,
	}
	if err := DB.Create(&clerk).Error; err != nil {
		log.Printf("seed role clerk error: %v", err)
	}

	// 操作工：报工权限 + 查看报工进度
	var workerPerms []model.Permission
	DB.Where("(resource = 'report' AND action IN ('create', 'read'))").Find(&workerPerms)

	worker := model.Role{
		Name:        "操作工",
		Description: "手机端扫码报工",
		Permissions: workerPerms,
	}
	if err := DB.Create(&worker).Error; err != nil {
		log.Printf("seed role worker error: %v", err)
	}
}

// ─── 用户 ────────────────────────────────────────────────────────────────────

func seedUsers() {
	var count int64
	DB.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	// 查询管理部 + 管理员角色
	var mgmtDept model.Department
	DB.Where("name = ?", "管理部").First(&mgmtDept)

	var adminRole model.Role
	DB.Where("name = ?", "管理员").First(&adminRole)

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("seed: hash password error: %v", err)
		return
	}

	admin := model.User{
		Name:         "admin",
		PasswordHash: string(hash),
		DepartmentID: &mgmtDept.ID,
		RoleID:       &adminRole.ID,
		IsActive:     true,
	}
	if err := DB.Create(&admin).Error; err != nil {
		log.Printf("seed user admin error: %v", err)
		return
	}
	log.Println("seed: default admin created  (name=admin  password=admin123)")
}

// ─── 工序模板 ─────────────────────────────────────────────────────────────────

func seedProcesses() {
	var count int64
	DB.Model(&model.Process{}).Count(&count)
	if count > 0 {
		return
	}

	names := []string{
		"下料", "折弯", "冲压", "焊接",
		"打磨", "喷漆", "组装", "检测", "包装",
	}
	for _, n := range names {
		p := model.Process{Name: n}
		if err := DB.Create(&p).Error; err != nil {
			log.Printf("seed process %s error: %v", n, err)
		}
	}
}
