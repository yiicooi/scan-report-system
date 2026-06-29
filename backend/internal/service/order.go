package service

import (
	"fmt"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/repository"
	"gorm.io/gorm"
)

// ScanOrder 扫码返回工单信息（含工序+汇总）
func ScanOrder(internalNo string) (*model.Order, error) {
	return repository.GetOrderWithProcesses(internalNo)
}

// ScanOrderForUser 扫码返回工单信息，工序列表只含该用户所属部门的工序
func ScanOrderForUser(internalNo string, userID uint) (*model.Order, error) {
	// 查用户部门
	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return repository.GetOrderWithProcesses(internalNo)
	}
	return repository.GetOrderWithProcessesForUser(internalNo, user.DepartmentID)
}

// CreateScrapOrder 创建报废子单，编号格式：主单号_BF01
func CreateScrapOrder(parentID uint, createdBy uint) (*model.Order, error) {
	var parent model.Order
	if err := database.DB.First(&parent, parentID).Error; err != nil {
		return nil, fmt.Errorf("主单不存在: %w", err)
	}
	if parent.OrderType != model.OrderTypeNormal {
		return nil, fmt.Errorf("只能对主单创建报废单")
	}

	var scrapOrder *model.Order
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 查找该主单下已有的报废单数量，生成 _BF 序号
		var count int64
		if err := tx.Model(&model.Order{}).
			Where("parent_id = ? AND order_type = ?", parentID, model.OrderTypeScrap).
			Count(&count).Error; err != nil {
			return err
		}
		internalNo := fmt.Sprintf("%s_BF%02d", parent.InternalNo, count+1)

		scrapOrder = &model.Order{
			InternalNo: internalNo,
			ExternalNo: parent.ExternalNo,
			DrawingNo:  parent.DrawingNo,
			DrawingURL: parent.DrawingURL,
			TotalQty:   0, // 待文员填写
			UnitPrice:  parent.UnitPrice,
			OrderDate:  parent.OrderDate,
			Status:     model.OrderStatusDraft,
			OrderType:  model.OrderTypeScrap,
			ParentID:   &parentID,
			CreatedBy:  createdBy,
		}
		return tx.Create(scrapOrder).Error
	})
	if err != nil {
		return nil, err
	}
	return scrapOrder, nil
}

// GetOrderProgress 获取工单报工进度
func GetOrderProgress(orderID uint) ([]map[string]interface{}, error) {
	var processes []model.OrderProcess
	err := database.DB.
		Preload("Summary").
		Preload("Process").
		Where("order_id = ?", orderID).
		Order("sort ASC").
		Find(&processes).Error
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(processes))
	for _, p := range processes {
		item := map[string]interface{}{
			"id":           p.ID,
			"display_name": p.DisplayName,
			"sort":         p.Sort,
			"status":       p.Status,
			"unit_hours":   p.UnitHours,
			"total_hours":  p.TotalHours,
			"deadline":     p.Deadline,
		}
		if p.Summary != nil {
			item["total_received"] = p.Summary.TotalReceived
			item["total_completed"] = p.Summary.TotalCompleted
			item["total_scrap"] = p.Summary.TotalScrap
			item["progress_pct"] = p.Summary.ProgressPct
			item["is_completed"] = p.Summary.IsCompleted
		}
		result = append(result, item)
	}
	return result, nil
}
