package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/service"
)

// ListUsers GET /api/admin/users
func ListUsers(c *gin.Context) {
	var users []model.User
	database.DB.Preload("Department").Preload("Role").Find(&users)
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// CreateUser POST /api/admin/users
func CreateUser(c *gin.Context) {
	var input struct {
		Name         string `json:"name" binding:"required"`
		Password     string `json:"password" binding:"required,min=6"`
		DepartmentID *uint  `json:"department_id"`
		RoleID       *uint  `json:"role_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, err := service.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user := model.User{
		Name:         input.Name,
		PasswordHash: hash,
		DepartmentID: input.DepartmentID,
		RoleID:       input.RoleID,
		IsActive:     true,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// GetUser GET /api/admin/users/:id
func GetUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := database.DB.Preload("Department").Preload("Role.Permissions").
		First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UpdateUser PUT /api/admin/users/:id
func UpdateUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var input struct {
		Name         string `json:"name"`
		Password     string `json:"password"`
		DepartmentID *uint  `json:"department_id"`
		RoleID       *uint  `json:"role_id"`
		IsActive     *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Name != "" {
		user.Name = input.Name
	}
	if input.Password != "" {
		hash, _ := service.HashPassword(input.Password)
		user.PasswordHash = hash
	}
	if input.DepartmentID != nil {
		user.DepartmentID = input.DepartmentID
	}
	if input.RoleID != nil {
		user.RoleID = input.RoleID
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}
	database.DB.Save(&user)
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// DeleteUser DELETE /api/admin/users/:id
func DeleteUser(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	database.DB.Delete(&model.User{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListDepartments GET /api/admin/departments
func ListDepartments(c *gin.Context) {
	var depts []model.Department
	database.DB.Where("parent_id IS NULL").
		Preload("Children").
		Preload("Children.Children").
		Preload("Children.Children.Children").
		Find(&depts)
	c.JSON(http.StatusOK, gin.H{"data": depts})
}

// CreateDepartment POST /api/admin/departments
func CreateDepartment(c *gin.Context) {
	var dept model.Department
	if err := c.ShouldBindJSON(&dept); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&dept)
	c.JSON(http.StatusCreated, gin.H{"data": dept})
}

// ListRoles GET /api/admin/roles
func ListRoles(c *gin.Context) {
	var roles []model.Role
	database.DB.Preload("Permissions").Find(&roles)
	c.JSON(http.StatusOK, gin.H{"data": roles})
}

// CreateRole POST /api/admin/roles
func CreateRole(c *gin.Context) {
	var role model.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&role)
	c.JSON(http.StatusCreated, gin.H{"data": role})
}

// UpdateRolePermissions PUT /api/admin/roles/:id/permissions
func UpdateRolePermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var role model.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var input struct {
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var permissions []model.Permission
	database.DB.Find(&permissions, input.PermissionIDs)
	database.DB.Model(&role).Association("Permissions").Replace(permissions)
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// ListPermissions GET /api/admin/permissions
func ListPermissions(c *gin.Context) {
	var perms []model.Permission
	database.DB.Find(&perms)
	c.JSON(http.StatusOK, gin.H{"data": perms})
}
