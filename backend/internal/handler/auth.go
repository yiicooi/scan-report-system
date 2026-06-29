package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/middleware"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/service"
)

type loginInput struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login POST /api/auth/login
func Login(c *gin.Context) {
	var input loginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	access, refresh, user, err := service.Login(input.Name, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	// 构建 permission 列表 ["resource:action", ...]
	perms := make([]string, 0, len(user.Role.Permissions))
	for _, p := range user.Role.Permissions {
		perms = append(perms, p.Resource+":"+p.Action)
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"user": gin.H{
			"id":          user.ID,
			"name":        user.Name,
			"role":        user.Role.Name,
			"permissions": perms,
		},
	})
}

// RefreshToken POST /api/auth/refresh
func RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims, err := middleware.ParseToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	var user model.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	access, refresh, err := middleware.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}
