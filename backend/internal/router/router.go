package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/handler"
	adminH "github.com/sui/scan-report/internal/handler/admin"
	appH "github.com/sui/scan-report/internal/handler/app"
	mcpH "github.com/sui/scan-report/internal/handler/mcp"
	"github.com/sui/scan-report/internal/middleware"
)

func Setup(r *gin.Engine) {
	r.Use(middleware.CORS())

	api := r.Group("/api")

	// ── 认证（无需 JWT）──────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
	}

	api.POST("/mcp", middleware.JWTAuth(), mcpH.Handle)

	// ── Admin（需 JWT）──────────────────────────────
	admin := api.Group("/admin", middleware.JWTAuth())
	{
		// 用户
		admin.GET("/users", adminH.ListUsers)
		admin.POST("/users", middleware.RequirePermission("user", "create"), adminH.CreateUser)
		admin.GET("/users/:id", adminH.GetUser)
		admin.PUT("/users/:id", middleware.RequirePermission("user", "update"), adminH.UpdateUser)
		admin.DELETE("/users/:id", middleware.RequirePermission("user", "delete"), adminH.DeleteUser)

		// 部门
		admin.GET("/departments", adminH.ListDepartments)
		admin.POST("/departments", middleware.RequirePermission("department", "create"), adminH.CreateDepartment)

		// 角色权限
		admin.GET("/roles", adminH.ListRoles)
		admin.POST("/roles", middleware.RequirePermission("role", "create"), adminH.CreateRole)
		admin.PUT("/roles/:id/permissions", middleware.RequirePermission("role", "update"), adminH.UpdateRolePermissions)
		admin.GET("/permissions", adminH.ListPermissions)

		// 工序模板
		admin.GET("/processes", adminH.ListProcesses)
		admin.POST("/processes", middleware.RequirePermission("process", "create"), adminH.CreateProcess)
		admin.PUT("/processes/:id", middleware.RequirePermission("process", "update"), adminH.UpdateProcess)
		admin.DELETE("/processes/:id", middleware.RequirePermission("process", "delete"), adminH.DeleteProcess)

		// 流程别名
		admin.GET("/process-aliases", adminH.ListProcessAliases)
		admin.POST("/process-aliases", middleware.RequirePermission("process", "create"), adminH.CreateProcessAlias)
		admin.GET("/process-aliases/:id", adminH.GetProcessAlias)
		admin.DELETE("/process-aliases/:id", middleware.RequirePermission("process", "delete"), adminH.DeleteProcessAlias)

		// 工单
		admin.GET("/orders", adminH.ListOrders)
		admin.POST("/orders", middleware.RequirePermission("order", "create"), adminH.CreateOrder)
		admin.POST("/orders/import-excel", middleware.RequirePermission("order", "create"), adminH.ImportOrdersExcel)
		admin.GET("/orders/:id", adminH.GetOrder)
		admin.PUT("/orders/:id", middleware.RequirePermission("order", "update"), adminH.UpdateOrder)
		admin.DELETE("/orders/:id", middleware.RequirePermission("order", "delete"), adminH.DeleteOrder)
		admin.POST("/orders/:id/scrap", middleware.RequirePermission("order", "create"), adminH.CreateScrapOrder)
		admin.GET("/orders/:id/scrap-orders", adminH.GetScrapOrders)
		admin.POST("/orders/:id/processes", middleware.RequirePermission("order", "update"), adminH.AddOrderProcess)
		admin.PUT("/orders/:id/processes/:pid", middleware.RequirePermission("order", "update"), adminH.UpdateOrderProcess)
		admin.GET("/orders/:id/report-progress", adminH.GetReportProgress)
		admin.GET("/order-processes/:id/report-details", adminH.GetOrderProcessReportDetails)

		// OSS 预签名
		admin.POST("/oss/presign", adminH.Presign)
	}

	// ── App（需 JWT）────────────────────────────────
	app := api.Group("/app", middleware.JWTAuth())
	{
		app.GET("/scan", appH.Scan)
		app.POST("/report", appH.SubmitReport)
		app.GET("/report/history", appH.ReportHistory)
		app.GET("/orders/:id/progress", appH.OrderProgress)
		app.GET("/ai/chat/messages", appH.AIChatMessages)
		app.POST("/ai/chat/stream", appH.AIChatStream)
		app.POST("/oss/presign", adminH.Presign) // 复用同一 handler
	}
}
