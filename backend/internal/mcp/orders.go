package mcp

import (
	"fmt"
	"strings"

	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
)

func QueryOrders(arguments map[string]interface{}) ([]model.Order, error) {
	orders, err := findOrders(arguments)
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func findOrders(arguments map[string]interface{}) ([]model.Order, error) {
	internalNo := strings.TrimSpace(stringArg(arguments, "internal_no"))
	externalNo := strings.TrimSpace(stringArg(arguments, "external_no"))
	partName := strings.TrimSpace(stringArg(arguments, "part_name"))
	if internalNo == "" && externalNo == "" && partName == "" {
		return nil, fmt.Errorf("请提供内部单号、外部单号或零件名称")
	}

	query := database.DB.Preload("Creator").Preload("Processes").Preload("Processes.Summary")
	if internalNo != "" {
		query = query.Where("internal_no LIKE ?", "%"+internalNo+"%")
	}
	if externalNo != "" {
		query = query.Where("external_no LIKE ?", "%"+externalNo+"%")
	}
	if partName != "" {
		conditions := []string{}
		values := []interface{}{}
		for _, candidate := range partNameCandidates(partName) {
			conditions = append(conditions, "part_name LIKE ?")
			values = append(values, "%"+candidate+"%")
		}
		query = query.Where(strings.Join(conditions, " OR "), values...)
	}

	var orders []model.Order
	if err := query.Order("created_at DESC").Limit(orderLimit(arguments)).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func partNameCandidates(partName string) []string {
	partName = strings.TrimSpace(partName)
	if partName == "" {
		return nil
	}

	seen := map[string]bool{}
	candidates := []string{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		candidates = append(candidates, v)
	}

	add(partName)
	for _, prefix := range []string{"大的", "小的", "大号", "小号", "大型", "小型", "大", "小"} {
		add(strings.TrimPrefix(partName, prefix))
	}
	for _, suffix := range []string{"大的", "小的", "大号", "小号", "大型", "小型"} {
		add(strings.TrimSuffix(partName, suffix))
	}
	return candidates
}
