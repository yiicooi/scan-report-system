package mcp

func queryOrderInputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"internal_no":  map[string]interface{}{"type": "string", "description": "内部单号，支持模糊查询"},
			"external_no":  map[string]interface{}{"type": "string", "description": "外部单号，支持模糊查询"},
			"part_name":    map[string]interface{}{"type": "string", "description": "零件名称，支持模糊查询"},
			"limit":        map[string]interface{}{"type": "number", "description": "返回工单数量上限，默认 5"},
			"detail_limit": map[string]interface{}{"type": "number", "description": "报工明细数量上限，默认 50，仅报工明细类工具使用"},
		},
	}
}
