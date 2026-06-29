package mcp

import "fmt"

func ListTools() []Tool {
	return []Tool{
		{
			Name:        "query_orders",
			Description: "按内部单号、外部单号或零件名称查询工单基础信息",
			InputSchema: queryOrderInputSchema(),
		},
		{
			Name:        "query_order_progress",
			Description: "按内部单号、外部单号或零件名称查询工单及工序进度",
			InputSchema: queryOrderInputSchema(),
		},
		{
			Name:        "query_order_report_details",
			Description: "查询工单每次报工明细，包括报工人、工序、数量、时间、备注和图片",
			InputSchema: queryOrderInputSchema(),
		},
		{
			Name:        "query_order_reporters",
			Description: "查询工单有哪些人参与报工，以及每个人参与的工序和数量汇总",
			InputSchema: queryOrderInputSchema(),
		},
	}
}

func CallTool(name string, arguments map[string]interface{}) (interface{}, error) {
	switch name {
	case "query_orders":
		return QueryOrders(arguments)
	case "query_order_progress":
		return QueryOrderProgress(arguments)
	case "query_order_report_details":
		return QueryOrderReportDetails(arguments)
	case "query_order_reporters":
		return QueryOrderReporters(arguments)
	default:
		return nil, fmt.Errorf("unknown MCP tool: %s", name)
	}
}
