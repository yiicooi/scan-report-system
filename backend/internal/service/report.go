package service

import (
	"fmt"
	"log"
	"time"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReportInput 提交报工入参
type ReportInput struct {
	OrderProcessID uint
	UserID         uint
	ReceivedQty    int
	CompletedQty   int
	ScrapQty       int
	ReceiveImages  model.StringSlice
	CompleteImages model.StringSlice
	ScrapImages    model.StringSlice
	Note           string
}

// ValidationError 业务校验失败，前端可直接展示 Msg
type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string { return e.Msg }

func validErr(format string, args ...interface{}) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

var (
	ErrAllZero        = &ValidationError{Msg: "接收、完成、报废数量不能同时为 0"}
	ErrExceedOrderQty = &ValidationError{Msg: "数量超出订单量"}
	ErrNegativeQty    = &ValidationError{Msg: "数量不能为负数"}
)

// SubmitReport 提交报工（带事务+行锁）
func SubmitReport(input ReportInput) (*model.WorkReportDetail, error) {
	if input.ReceivedQty < 0 || input.CompletedQty < 0 || input.ScrapQty < 0 {
		return nil, ErrNegativeQty
	}
	if input.ReceivedQty == 0 && input.CompletedQty == 0 && input.ScrapQty == 0 {
		return nil, ErrAllZero
	}

	var detail *model.WorkReportDetail

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. 获取工序信息和工单数量
		var op model.OrderProcess
		if err := tx.First(&op, input.OrderProcessID).Error; err != nil {
			return fmt.Errorf("工序不存在: %w", err)
		}
		var order model.Order
		if err := tx.First(&order, op.OrderID).Error; err != nil {
			return err
		}

		// 2. 行锁：获取或创建汇总记录
		var summary model.WorkReportSummary
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_process_id = ?", input.OrderProcessID).
			First(&summary)
		if result.Error != nil {
			// 首次报工，创建汇总行
			summary = model.WorkReportSummary{OrderProcessID: input.OrderProcessID}
			if err := tx.Create(&summary).Error; err != nil {
				return err
			}
			// 重新锁定
			tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("order_process_id = ?", input.OrderProcessID).
				First(&summary)
		}

		// 3. 数量校验
		newReceived := summary.TotalReceived + input.ReceivedQty
		newCompleted := summary.TotalCompleted + input.CompletedQty
		newScrap := summary.TotalScrap + input.ScrapQty

		log.Printf("[report] orderID=%d totalQty=%d input(r=%d c=%d s=%d) summary(r=%d c=%d s=%d)",
			order.ID, order.TotalQty,
			input.ReceivedQty, input.CompletedQty, input.ScrapQty,
			summary.TotalReceived, summary.TotalCompleted, summary.TotalScrap)

		if order.TotalQty > 0 {
			remaining := order.TotalQty - summary.TotalReceived
			if newReceived > order.TotalQty {
				return validErr("接收数量超出订单量，当前已接收 %d，剩余可接收 %d", summary.TotalReceived, remaining)
			}
			if newCompleted > order.TotalQty {
				return validErr("完成数量超出订单量，订单总量 %d，当前已完成 %d", order.TotalQty, summary.TotalCompleted)
			}
			if newScrap > order.TotalQty {
				return validErr("报废数量超出订单量，订单总量 %d，当前已报废 %d", order.TotalQty, summary.TotalScrap)
			}
		}

		// 上一道工序校验：累计接收数量不可超过上一道工序的完成数量（首道不检查）
		if newReceived > 0 && op.Sort > 1 {
			var prevOP model.OrderProcess
			err := tx.Where("order_id = ? AND sort < ? AND sort > 0", op.OrderID, op.Sort).
				Order("sort DESC").First(&prevOP).Error
			if err == nil {
				var prevSummary model.WorkReportSummary
				if err2 := tx.Where("order_process_id = ?", prevOP.ID).First(&prevSummary).Error; err2 == nil {
					if newReceived > prevSummary.TotalCompleted {
						return validErr("累计接收数量（%d）不可大于上一道工序「%s」的完成数量（%d）",
							newReceived, prevOP.DisplayName, prevSummary.TotalCompleted)
					}
				} else {
					return validErr("上一道工序「%s」尚未有完成记录，请先完成上一道工序", prevOP.DisplayName)
				}
			}
		}

		// 完成数量不可超过累计接收数量
		if newCompleted > newReceived {
			return validErr("完成数量（%d）不可大于已接收数量（%d）", newCompleted, newReceived)
		}
		// 报废+完成不可超过接收
		if newScrap+newCompleted > newReceived {
			return validErr("报废数量（%d）+ 完成数量（%d）不可大于已接收数量（%d）", newScrap, newCompleted, newReceived)
		}

		// 4. 插入明细
		detail = &model.WorkReportDetail{
			SummaryID:      summary.ID,
			OrderProcessID: input.OrderProcessID,
			UserID:         input.UserID,
			ReceivedQty:    input.ReceivedQty,
			CompletedQty:   input.CompletedQty,
			ScrapQty:       input.ScrapQty,
			ReceiveImages:  input.ReceiveImages,
			CompleteImages: input.CompleteImages,
			ScrapImages:    input.ScrapImages,
			Note:           input.Note,
			ReportedAt:     time.Now(),
		}
		if err := tx.Create(detail).Error; err != nil {
			return err
		}

		// 5. 更新汇总表
		isCompleted := newCompleted >= order.TotalQty
		progressPct := 0.0
		if order.TotalQty > 0 {
			progressPct = float64(newCompleted) / float64(order.TotalQty) * 100
		}

		if err := tx.Model(&summary).Updates(map[string]interface{}{
			"total_received":  newReceived,
			"total_completed": newCompleted,
			"total_scrap":     newScrap,
			"progress_pct":    progressPct,
			"is_completed":    isCompleted,
		}).Error; err != nil {
			return err
		}

		// 6. 更新工序状态
		newStatus := model.ProcessStatusInProgress
		if isCompleted {
			newStatus = model.ProcessStatusCompleted
		}
		if err := tx.Model(&op).Update("status", newStatus).Error; err != nil {
			return err
		}

		if order.Status == model.OrderStatusDraft || order.Status == model.OrderStatusReady {
			if err := tx.Model(&order).Update("status", model.OrderStatusActive).Error; err != nil {
				return err
			}
			order.Status = model.OrderStatusActive
		}

		// 7. 若是最后一道工序：同步订单 total_completed
		var maxSort int
		tx.Model(&model.OrderProcess{}).Where("order_id = ?", op.OrderID).
			Select("MAX(sort)").Scan(&maxSort)

		if op.Sort == maxSort {
			if err := tx.Model(&order).Update("total_completed", newCompleted).Error; err != nil {
				return err
			}
			// 若已完成，检查是否可以将订单置为 completed
			if isCompleted {
				if err := tryCompleteOrder(tx, &order, newCompleted); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return detail, err
}

// tryCompleteOrder 联合判断主单+报废单是否都完成
func tryCompleteOrder(tx *gorm.DB, order *model.Order, currentCompleted int) error {
	var mainOrder model.Order
	mainID := order.ID
	if order.OrderType == model.OrderTypeScrap {
		// 报废单完成后，找主单重新判断
		if err := tx.First(&mainOrder, order.ParentID).Error; err != nil {
			return err
		}
		mainID = mainOrder.ID
	} else {
		mainOrder = *order
	}

	// 主单最后工序完成数
	var mainLastCompleted int
	tx.Model(&model.WorkReportSummary{}).
		Joins("JOIN order_processes ON order_processes.id = work_report_summaries.order_process_id").
		Where("order_processes.order_id = ? AND order_processes.sort = (SELECT MAX(sort) FROM order_processes WHERE order_id = ?)",
			mainID, mainID).
		Select("COALESCE(total_completed, 0)").
		Scan(&mainLastCompleted)

	// 所有报废单最后工序完成数之和
	var scrapTotal int
	tx.Model(&model.WorkReportSummary{}).
		Joins("JOIN order_processes ON order_processes.id = work_report_summaries.order_process_id").
		Joins("JOIN orders ON orders.id = order_processes.order_id").
		Where("orders.parent_id = ? AND order_processes.sort = (SELECT MAX(sort) FROM order_processes WHERE order_id = orders.id)",
			mainID).
		Select("COALESCE(SUM(work_report_summaries.total_completed), 0)").
		Scan(&scrapTotal)

	// 所有报废单是否都完成
	var incompleteScraps int64
	tx.Model(&model.Order{}).
		Where("parent_id = ? AND order_type = ? AND status != ?", mainID, model.OrderTypeScrap, model.OrderStatusCompleted).
		Count(&incompleteScraps)

	// 联合判断
	if mainLastCompleted+scrapTotal >= mainOrder.TotalQty && incompleteScraps == 0 {
		// 标记报废单为 completed（若是当前单）
		if order.OrderType == model.OrderTypeScrap {
			if err := tx.Model(order).Update("status", model.OrderStatusCompleted).Error; err != nil {
				return err
			}
		}
		// 标记主单为 completed
		return tx.Model(&mainOrder).Update("status", model.OrderStatusCompleted).Error
	}

	// 若只是报废单完成但主单未满足条件，单独置报废单 completed
	if order.OrderType == model.OrderTypeScrap {
		return tx.Model(order).Update("status", model.OrderStatusCompleted).Error
	}

	return nil
}
