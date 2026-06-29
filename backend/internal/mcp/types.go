package mcp

import (
	"time"

	"github.com/sui/scan-report/internal/model"
)

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type OrderProgressResult struct {
	Order    model.Order              `json:"order"`
	Progress []map[string]interface{} `json:"progress"`
}

type OrderReportDetailsResult struct {
	Order   model.Order        `json:"order"`
	Details []ReportDetailItem `json:"details"`
}

type ReportDetailItem struct {
	ID             uint      `json:"id"`
	OrderProcessID uint      `json:"order_process_id"`
	ProcessName    string    `json:"process_name"`
	UserID         uint      `json:"user_id"`
	UserName       string    `json:"user_name"`
	DepartmentName string    `json:"department_name"`
	ReceivedQty    int       `json:"received_qty"`
	CompletedQty   int       `json:"completed_qty"`
	ScrapQty       int       `json:"scrap_qty"`
	Note           string    `json:"note"`
	ReportedAt     time.Time `json:"reported_at"`
	ReceiveImages  []string  `json:"receive_images,omitempty"`
	CompleteImages []string  `json:"complete_images,omitempty"`
	ScrapImages    []string  `json:"scrap_images,omitempty"`
}

type OrderReportersResult struct {
	Order     model.Order    `json:"order"`
	Reporters []ReporterItem `json:"reporters"`
}

type ReporterItem struct {
	UserID         uint      `json:"user_id"`
	UserName       string    `json:"user_name"`
	DepartmentName string    `json:"department_name"`
	ProcessNames   []string  `json:"process_names"`
	TotalReceived  int       `json:"total_received"`
	TotalCompleted int       `json:"total_completed"`
	TotalScrap     int       `json:"total_scrap"`
	ReportCount    int       `json:"report_count"`
	LastReportedAt time.Time `json:"last_reported_at"`
}
