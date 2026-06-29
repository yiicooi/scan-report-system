package mcp

import "github.com/sui/scan-report/internal/service"

func QueryOrderProgress(arguments map[string]interface{}) ([]OrderProgressResult, error) {
	orders, err := findOrders(arguments)
	if err != nil {
		return nil, err
	}

	results := make([]OrderProgressResult, 0, len(orders))
	for _, order := range orders {
		progress, err := service.GetOrderProgress(order.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, OrderProgressResult{
			Order:    order,
			Progress: progress,
		})
	}
	return results, nil
}
