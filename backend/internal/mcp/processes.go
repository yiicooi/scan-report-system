package mcp

import (
	"strings"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
)

func findOrderProcesses(orderID uint) ([]model.OrderProcess, error) {
	var processes []model.OrderProcess
	if err := database.DB.
		Preload("Process").
		Where("order_id = ?", orderID).
		Order("sort ASC").
		Find(&processes).Error; err != nil {
		return nil, err
	}
	return processes, nil
}

func processNameMap(processes []model.OrderProcess) map[uint]string {
	result := map[uint]string{}
	for _, process := range processes {
		name := strings.TrimSpace(process.DisplayName)
		if name == "" {
			name = process.Process.Name
		}
		result[process.ID] = name
	}
	return result
}
