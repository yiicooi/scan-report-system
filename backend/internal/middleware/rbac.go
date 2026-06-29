package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
)

// RequirePermission RBAC 权限校验，resource:action 如 "order:create"
func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, exists := c.Get("role_id")
		if !exists || roleID.(uint) == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no role assigned"})
			return
		}

		var count int64
		database.DB.Model(&model.Permission{}).
			Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
			Where("role_permissions.role_id = ? AND permissions.resource = ? AND permissions.action IN (?, ?)",
				roleID, resource, action, "*").
			Count(&count)

		if count == 0 {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": fmt.Sprintf("permission denied: %s:%s", resource, action)})
			return
		}
		c.Next()
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
