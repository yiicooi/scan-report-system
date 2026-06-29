package model

import "time"

type OrderStatus string
type OrderType string
type OrderProcessStatus string

const (
	OrderStatusDraft     OrderStatus = "draft"
	OrderStatusReady     OrderStatus = "ready"
	OrderStatusActive    OrderStatus = "active"
	OrderStatusCompleted OrderStatus = "completed"

	OrderTypeNormal OrderType = "normal"
	OrderTypeScrap  OrderType = "scrap"

	ProcessStatusPending    OrderProcessStatus = "pending"
	ProcessStatusInProgress OrderProcessStatus = "in_progress"
	ProcessStatusCompleted  OrderProcessStatus = "completed"
)

// Order 工单
type Order struct {
	BaseModel
	InternalNo     string         `gorm:"size:50;uniqueIndex;not null" json:"internal_no"`
	ExternalNo     string         `gorm:"size:50" json:"external_no"`
	PartName       string         `gorm:"size:200" json:"part_name"`
	DrawingNo      string         `gorm:"size:100" json:"drawing_no"`
	DrawingURL     string         `gorm:"size:500" json:"drawing_url"`
	TotalQty       int            `gorm:"not null;default:0" json:"total_qty"`
	UnitPrice      float64        `gorm:"type:decimal(12,4);default:0" json:"unit_price"`
	TotalAmount    float64        `gorm:"type:decimal(14,4);default:0" json:"total_amount"`
	OrderDate      time.Time      `json:"order_date"`
	Status         OrderStatus    `gorm:"size:20;default:'draft'" json:"status"`
	OrderType      OrderType      `gorm:"size:20;default:'normal'" json:"order_type"`
	ParentID       *uint          `json:"parent_id"` // 报废单关联主单
	TotalCompleted int            `gorm:"default:0" json:"total_completed"`
	CreatedBy      uint           `json:"created_by"`
	Creator        *User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Processes      []OrderProcess `gorm:"foreignKey:OrderID" json:"processes,omitempty"`
	ScrapOrders    []Order        `gorm:"foreignKey:ParentID" json:"scrap_orders,omitempty"`
}

// OrderProcess 工单工序明细
type OrderProcess struct {
	BaseModel
	OrderID     uint               `gorm:"not null;index" json:"order_id"`
	ProcessID   uint               `gorm:"not null" json:"process_id"`
	Process     Process            `gorm:"foreignKey:ProcessID" json:"process,omitempty"`
	DisplayName string             `gorm:"size:100;not null" json:"display_name"`
	Sort        int                `gorm:"not null;default:0" json:"sort"`
	UnitHours   float64            `gorm:"type:decimal(8,2);default:0" json:"unit_hours"`
	TotalHours  float64            `gorm:"type:decimal(10,2);default:0" json:"total_hours"`
	Deadline    *time.Time         `json:"deadline"`
	Status      OrderProcessStatus `gorm:"size:20;default:'pending'" json:"status"`
	Summary     *WorkReportSummary `gorm:"foreignKey:OrderProcessID" json:"summary,omitempty"`
	CanReport   bool               `gorm:"-" json:"can_report"` // 当前用户是否可在此报工（不存表）
}
