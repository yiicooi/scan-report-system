package database

import (
	"fmt"
	"log"

	"github.com/sui/scan-report/config"
	"github.com/sui/scan-report/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := config.Cfg.Database.DSN
	logLevel := logger.Info
	if config.Cfg.Server.Mode == "release" {
		logLevel = logger.Error
	}
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	log.Println("database connected")
}

func AutoMigrate() {
	err := DB.AutoMigrate(
		&model.Department{},
		&model.Role{},
		&model.Permission{},
		&model.User{},
		&model.Process{},
		&model.ProcessAlias{},
		&model.ProcessAliasItem{},
		&model.Order{},
		&model.OrderProcess{},
		&model.WorkReportSummary{},
		&model.WorkReportDetail{},
		&model.AIChatSession{},
		&model.AIChatMessage{},
	)
	if err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}
	log.Println("database migrated")
}

func RunSQLMigrations() {
	sqls := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_role_permissions ON role_permissions(role_id, permission_id)`,
		`CREATE INDEX IF NOT EXISTS idx_order_processes_order_id_sort ON order_processes(order_id, sort)`,
		`CREATE INDEX IF NOT EXISTS idx_work_report_details_reported_at ON work_report_details(reported_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_chat_sessions_user_updated ON ai_chat_sessions(user_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_session_created ON ai_chat_messages(session_id, created_at)`,
		`ALTER TABLE work_report_details DROP CONSTRAINT IF EXISTS chk_qty_not_all_zero`,
		`ALTER TABLE work_report_details ADD CONSTRAINT chk_qty_not_all_zero
CHECK (received_qty > 0 OR completed_qty > 0 OR scrap_qty > 0)`,
		`ALTER TABLE work_report_details DROP CONSTRAINT IF EXISTS chk_qty_non_negative`,
		`ALTER TABLE work_report_details ADD CONSTRAINT chk_qty_non_negative
CHECK (received_qty >= 0 AND completed_qty >= 0 AND scrap_qty >= 0)`,
		`UPDATE orders
SET status = 'active'
WHERE status IN ('draft', 'ready')
AND EXISTS (
	SELECT 1
	FROM order_processes
	JOIN work_report_details ON work_report_details.order_process_id = order_processes.id
	WHERE order_processes.order_id = orders.id
)`,
		`UPDATE orders
SET status = 'ready'
WHERE status = 'draft'
AND EXISTS (
	SELECT 1
	FROM order_processes
	WHERE order_processes.order_id = orders.id
)`,
	}
	for _, sql := range sqls {
		if err := DB.Exec(sql).Error; err != nil {
			fmt.Printf("migration warning: %v\n", err)
		}
	}
	log.Println("sql migrations applied")
}
