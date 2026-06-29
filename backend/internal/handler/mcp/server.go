package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	mcpTools "github.com/sui/scan-report/internal/mcp"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Handle(c *gin.Context) {
	var req rpcRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, rpcResponse{
			JSONRPC: "2.0",
			Error:   &rpcError{Code: -32700, Message: err.Error()},
		})
		return
	}

	result, rpcErr := dispatch(req.Method, req.Params)
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	c.JSON(http.StatusOK, resp)
}

func dispatch(method string, raw json.RawMessage) (interface{}, *rpcError) {
	switch method {
	case "initialize":
		return map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]interface{}{
				"name":    "scan-report-mcp",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		}, nil
	case "tools/list":
		return map[string]interface{}{"tools": mcpTools.ListTools()}, nil
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, &rpcError{Code: -32602, Message: err.Error()}
		}
		result, err := mcpTools.CallTool(params.Name, params.Arguments)
		if err != nil {
			return nil, &rpcError{Code: -32000, Message: err.Error()}
		}
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "json",
					"json": result,
				},
			},
		}, nil
	default:
		return nil, &rpcError{Code: -32601, Message: "method not found"}
	}
}
