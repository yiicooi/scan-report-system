package repository

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"gorm.io/gorm"
)

func GetOrderWithProcesses(internalNo string) (*model.Order, error) {
	var order model.Order
	err := database.DB.
		Preload("Processes.Summary").
		Preload("Processes.Process").
		Where("internal_no = ?", internalNo).
		First(&order).Error
	return &order, err
}

// GetOrderWithProcessesForUser 扫码查询，返回全部工序（含 summary），并标记当前部门可报工的工序
func GetOrderWithProcessesForUser(internalNo string, deptID *uint) (*model.Order, error) {
	var order model.Order
	err := database.DB.
		Where("internal_no = ?", internalNo).
		First(&order).Error
	if err != nil {
		return nil, err
	}

	// 拉取全部工序，按 sort 排序
	var processes []model.OrderProcess
	if err := database.DB.
		Preload("Summary").
		Preload("Process").
		Where("order_id = ?", order.ID).
		Order("sort ASC").
		Find(&processes).Error; err != nil {
		return nil, err
	}

	// 标记哪些工序属于当前部门（可报工）
	for i := range processes {
		if deptID == nil {
			processes[i].CanReport = true
		} else {
			dID := processes[i].Process.DepartmentID
			processes[i].CanReport = (dID == nil || *dID == *deptID)
		}
	}

	order.Processes = processes
	return &order, nil
}

func GetOrCreateSummary(orderProcessID uint) (*model.WorkReportSummary, error) {
	var summary model.WorkReportSummary
	result := database.DB.Where("order_process_id = ?", orderProcessID).First(&summary)
	if result.Error != nil {
		summary = model.WorkReportSummary{OrderProcessID: orderProcessID}
		if err := database.DB.Create(&summary).Error; err != nil {
			return nil, err
		}
	}
	return &summary, nil
}

func GetScrapOrdersByParentID(parentID uint) ([]model.Order, error) {
	var orders []model.Order
	err := database.DB.Where("parent_id = ? AND order_type = ?", parentID, model.OrderTypeScrap).Find(&orders).Error
	return orders, err
}

func CountScrapOrders(parentID uint) (int64, error) {
	var count int64
	err := database.DB.Model(&model.Order{}).
		Where("parent_id = ? AND order_type = ?", parentID, model.OrderTypeScrap).
		Count(&count).Error
	return count, err
}

// GenerateInternalNo 生成内部单号，格式：SC_YYYYMM0001（主单和报废单共用同一序号池）
func GenerateInternalNo(tx *gorm.DB, now time.Time) (string, error) {
	prefix := fmt.Sprintf("SC_%s", now.Format("200601"))
	var last model.Order
	if err := tx.Select("internal_no").
		Where("internal_no LIKE ?", prefix+"%").
		Order("internal_no DESC").
		First(&last).Error; err != nil && err != gorm.ErrRecordNotFound {
		return "", err
	}
	seq := 1
	if last.InternalNo != "" {
		if n, convErr := strconv.Atoi(last.InternalNo[len(prefix):]); convErr == nil && n >= 0 {
			seq = n + 1
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}
