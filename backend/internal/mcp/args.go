package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func stringArg(args map[string]interface{}, key string) string {
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

func intArg(args map[string]interface{}, key string, fallback int) int {
	v, ok := args[key]
	if !ok || v == nil {
		return fallback
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case json.Number:
		n, err := strconv.Atoi(t.String())
		if err == nil {
			return n
		}
	case string:
		n, err := strconv.Atoi(t)
		if err == nil {
			return n
		}
	}
	return fallback
}

func orderLimit(arguments map[string]interface{}) int {
	limit := intArg(arguments, "limit", 5)
	if limit <= 0 || limit > 20 {
		return 5
	}
	return limit
}

func detailLimit(arguments map[string]interface{}) int {
	limit := intArg(arguments, "detail_limit", 50)
	if limit <= 0 || limit > 200 {
		return 50
	}
	return limit
}
