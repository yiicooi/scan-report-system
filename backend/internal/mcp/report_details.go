package mcp

import (
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
)

func QueryOrderReportDetails(arguments map[string]interface{}) ([]OrderReportDetailsResult, error) {
	orders, err := findOrders(arguments)
	if err != nil {
		return nil, err
	}

	results := make([]OrderReportDetailsResult, 0, len(orders))
	for _, order := range orders {
		processes, err := findOrderProcesses(order.ID)
		if err != nil {
			return nil, err
		}
		processNames := processNameMap(processes)
		processIDs := make([]uint, 0, len(processes))
		for _, p := range processes {
			processIDs = append(processIDs, p.ID)
		}

		items := []ReportDetailItem{}
		if len(processIDs) > 0 {
			var details []model.WorkReportDetail
			if err := database.DB.
				Preload("User").
				Preload("User.Department").
				Where("order_process_id IN ?", processIDs).
				Order("reported_at DESC").
				Limit(detailLimit(arguments)).
				Find(&details).Error; err != nil {
				return nil, err
			}
			for _, detail := range details {
				items = append(items, buildReportDetailItem(detail, processNames))
			}
		}

		results = append(results, OrderReportDetailsResult{
			Order:   order,
			Details: items,
		})
	}
	return results, nil
}

func buildReportDetailItem(detail model.WorkReportDetail, processNames map[uint]string) ReportDetailItem {
	userName := ""
	departmentName := ""
	if detail.User != nil {
		userName = detail.User.Name
		if detail.User.Department != nil {
			departmentName = detail.User.Department.Name
		}
	}
	return ReportDetailItem{
		ID:             detail.ID,
		OrderProcessID: detail.OrderProcessID,
		ProcessName:    processNames[detail.OrderProcessID],
		UserID:         detail.UserID,
		UserName:       userName,
		DepartmentName: departmentName,
		ReceivedQty:    detail.ReceivedQty,
		CompletedQty:   detail.CompletedQty,
		ScrapQty:       detail.ScrapQty,
		Note:           detail.Note,
		ReportedAt:     detail.ReportedAt,
		ReceiveImages:  []string(detail.ReceiveImages),
		CompleteImages: []string(detail.CompleteImages),
		ScrapImages:    []string(detail.ScrapImages),
	}
}
