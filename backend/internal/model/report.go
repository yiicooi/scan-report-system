package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// StringSlice JSON 字符串数组
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(val interface{}) error {
	var str string
	switch v := val.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
	return json.Unmarshal([]byte(str), s)
}

// WorkReportSummary 报工汇总表（每道工序恰好一条）
type WorkReportSummary struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	OrderProcessID uint      `gorm:"uniqueIndex;not null" json:"order_process_id"`
	TotalReceived  int       `gorm:"default:0" json:"total_received"`
	TotalCompleted int       `gorm:"default:0" json:"total_completed"`
	TotalScrap     int       `gorm:"default:0" json:"total_scrap"`
	ProgressPct    float64   `gorm:"type:decimal(5,2);default:0" json:"progress_pct"` // = total_completed / order.total_qty * 100
	IsCompleted    bool      `gorm:"default:false" json:"is_completed"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// WorkReportDetail 报工明细表（一行多列：每次提交一条记录）
type WorkReportDetail struct {
	ID             uint        `gorm:"primarykey" json:"id"`
	SummaryID      uint        `gorm:"not null;index" json:"summary_id"`
	OrderProcessID uint        `gorm:"not null;index" json:"order_process_id"`
	UserID         uint        `gorm:"not null;index" json:"user_id"`
	User           *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ReceivedQty    int         `gorm:"default:0" json:"received_qty"`   // 本次接收，可为 0
	CompletedQty   int         `gorm:"default:0" json:"completed_qty"`  // 本次完成，可为 0
	ScrapQty       int         `gorm:"default:0" json:"scrap_qty"`      // 本次报废，可为 0
	ReceiveImages  StringSlice `gorm:"type:text" json:"receive_images"` // OSS URLs
	CompleteImages StringSlice `gorm:"type:text" json:"complete_images"`
	ScrapImages    StringSlice `gorm:"type:text" json:"scrap_images"`
	Note           string      `gorm:"size:500" json:"note"`
	ReportedAt     time.Time   `json:"reported_at"`
}
