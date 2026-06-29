package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/config"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/pkg/oss"
	"github.com/sui/scan-report/internal/router"
)

func main() {
	// 加载配置
	config.Load()

	// 连接数据库
	database.Connect()

	// AutoMigrate + SQL 补丁
	database.AutoMigrate()
	database.RunSQLMigrations()

	// 写入初始种子数据（幂等，有数据则跳过）
	database.Seed()

	// 初始化 OSS
	if err := oss.Init(); err != nil {
		log.Printf("OSS init warning: %v", err)
	}

	// 创建 Gin 引擎
	gin.SetMode(config.Cfg.Server.Mode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 注册路由
	router.Setup(r)

	addr := ":" + config.Cfg.Server.Port
	log.Printf("server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
