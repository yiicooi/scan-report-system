package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/config"
	mcpTools "github.com/sui/scan-report/internal/mcp"
)

type aiChatInput struct {
	Message string `json:"message" binding:"required"`
}

type deepSeekChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type deepSeekCompletion struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type extractedQuery struct {
	Intent     string  `json:"intent"`
	InternalNo string  `json:"internal_no"`
	ExternalNo string  `json:"external_no"`
	PartName   string  `json:"part_name"`
	Confidence float64 `json:"confidence"`
}

type aiQueryContext struct {
	Args      map[string]interface{}
	UpdatedAt time.Time
}

var aiQueryContexts sync.Map

const aiQueryContextTTL = 30 * time.Minute

// AIChatStream POST /api/app/ai/chat/stream
func AIChatStream(c *gin.Context) {
	var input aiChatInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	session, err := getOrCreateAIChatSession(userID)
	if err != nil {
		streamError(c, err.Error())
		return
	}
	createAIChatMessage(session.ID, "user", input.Message, "", nil, nil)

	toolName, args := buildToolCall(c.Request.Context(), input.Message)
	args = applyPreviousQueryContext(userID, args)
	result, err := mcpTools.CallTool(toolName, args)
	if err != nil {
		streamError(c, err.Error())
		return
	}
	saveQueryContext(userID, args)
	updateAIChatSessionQueryArgs(session.ID, args)

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	if config.Cfg.AI.DeepSeekAPIKey == "" {
		answer := streamFallbackAnswer(c, input.Message, toolName, result)
		createAIChatMessage(session.ID, "assistant", answer, toolName, args, result)
		return
	}

	answer, err := streamDeepSeekAnswer(c, input.Message, toolName, result)
	if err != nil {
		writeSSE(c.Writer, "error", err.Error())
	}
	createAIChatMessage(session.ID, "assistant", answer, toolName, args, result)
	writeSSE(c.Writer, "done", "")
}

func buildToolCall(ctx context.Context, message string) (string, map[string]interface{}) {
	if args, ok := extractRuleToolArgs(message); ok {
		return selectToolName(message, ""), args
	}

	if config.Cfg.AI.DeepSeekAPIKey != "" {
		if extracted, err := extractQueryWithDeepSeek(ctx, message); err == nil {
			if args, ok := toolArgsFromExtraction(extracted); ok {
				return selectToolName(message, extracted.Intent), args
			}
		}
	}

	return selectToolName(message, ""), extractFallbackToolArgs(message)
}

func applyPreviousQueryContext(userID uint, args map[string]interface{}) map[string]interface{} {
	if hasQueryTarget(args) {
		return args
	}
	if userID == 0 {
		return args
	}
	raw, ok := aiQueryContexts.Load(userID)
	if !ok {
		return applyStoredQueryContext(userID, args)
	}
	previous, ok := raw.(aiQueryContext)
	if !ok || time.Since(previous.UpdatedAt) > aiQueryContextTTL {
		aiQueryContexts.Delete(userID)
		return applyStoredQueryContext(userID, args)
	}
	for key, value := range previous.Args {
		if key == "internal_no" || key == "external_no" || key == "part_name" {
			args[key] = value
		}
	}
	return args
}

func applyStoredQueryContext(userID uint, args map[string]interface{}) map[string]interface{} {
	if previous := loadLastQueryArgs(userID); hasQueryTarget(previous) {
		for key, value := range previous {
			if key == "internal_no" || key == "external_no" || key == "part_name" {
				args[key] = value
			}
		}
	}
	return args
}

func saveQueryContext(userID uint, args map[string]interface{}) {
	if userID == 0 || !hasQueryTarget(args) {
		return
	}
	remembered := map[string]interface{}{"limit": 5}
	for _, key := range []string{"internal_no", "external_no", "part_name"} {
		if v := strings.TrimSpace(stringValue(args[key])); v != "" {
			remembered[key] = v
		}
	}
	aiQueryContexts.Store(userID, aiQueryContext{
		Args:      remembered,
		UpdatedAt: time.Now(),
	})
}

func hasQueryTarget(args map[string]interface{}) bool {
	for _, key := range []string{"internal_no", "external_no", "part_name"} {
		if strings.TrimSpace(stringValue(args[key])) != "" {
			return true
		}
	}
	return false
}

func stringValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func buildToolArgs(ctx context.Context, message string) map[string]interface{} {
	_, args := buildToolCall(ctx, message)
	return args
}

func extractRuleToolArgs(message string) (map[string]interface{}, bool) {
	args := map[string]interface{}{"limit": 5}
	if m := regexp.MustCompile(`SC_[A-Za-z0-9_]+`).FindString(message); m != "" {
		args["internal_no"] = m
		return args, true
	}

	if v := extractAfterLabel(message, `外部单号|外部订单号|客户单号`); v != "" {
		args["external_no"] = v
		return args, true
	}
	if v := extractAfterLabel(message, `零件名称|零件|产品|品名`); v != "" {
		args["part_name"] = v
		return args, true
	}
	return args, false
}

func extractFallbackToolArgs(message string) map[string]interface{} {
	args := map[string]interface{}{"limit": 5}
	cleaned := cleanProgressKeyword(message)
	if cleaned != "" {
		args["part_name"] = cleaned
	}
	return args
}

func toolArgsFromExtraction(extracted extractedQuery) (map[string]interface{}, bool) {
	args := map[string]interface{}{"limit": 5}
	intent := strings.TrimSpace(extracted.Intent)
	if intent == "unknown" {
		return args, false
	}
	if v := cleanProgressKeyword(extracted.InternalNo); v != "" {
		args["internal_no"] = v
		return args, true
	}
	if v := cleanProgressKeyword(extracted.ExternalNo); v != "" {
		args["external_no"] = v
		return args, true
	}
	if v := cleanProgressKeyword(extracted.PartName); v != "" {
		args["part_name"] = v
		return args, true
	}
	return args, false
}

func selectToolName(message, intent string) string {
	switch strings.TrimSpace(intent) {
	case "query_orders", "query_order_progress", "query_order_report_details", "query_order_reporters":
		return intent
	}

	reporterKeywords := []string{"谁做", "谁干", "谁报", "谁操作", "谁加工", "谁完成", "谁接收", "谁报工", "哪位", "哪个人", "哪些人", "报工人", "操作员", "员工", "工人", "人员"}
	for _, keyword := range reporterKeywords {
		if strings.Contains(message, keyword) {
			return "query_order_reporters"
		}
	}

	detailKeywords := []string{"报工明细", "报工详情", "报工记录", "明细", "详情", "记录", "图片", "备注"}
	for _, keyword := range detailKeywords {
		if strings.Contains(message, keyword) {
			return "query_order_report_details"
		}
	}

	orderKeywords := []string{"工单信息", "订单信息", "单据信息", "基础信息", "图纸", "图号", "图纸编号", "外部单号", "内部单号", "数量", "金额", "价格"}
	for _, keyword := range orderKeywords {
		if strings.Contains(message, keyword) {
			return "query_orders"
		}
	}

	return "query_order_progress"
}

func extractAfterLabel(message, labelPattern string) string {
	re := regexp.MustCompile(`(?:` + labelPattern + `)\s*[:：]?\s*([\p{Han}A-Za-z0-9_-]+)`)
	matches := re.FindStringSubmatch(message)
	if len(matches) < 2 {
		return ""
	}
	return cleanProgressKeyword(matches[1])
}

func cleanProgressKeyword(message string) string {
	cleaned := strings.TrimSpace(message)
	cleaned = strings.Trim(cleaned, "，。？！? !")
	replacements := []string{
		"帮我", "麻烦", "请", "查一下", "查询一下", "查下", "查询", "看一下", "看下",
		"进度", "报工", "工单", "订单", "零件", "零件名称", "产品", "品名",
		"现在", "目前", "已经", "有没有", "是否",
		"谁做的", "谁做", "谁干的", "谁干", "谁报工的", "谁报工", "谁报的", "谁报",
		"谁操作的", "谁操作", "谁加工的", "谁加工", "哪些人做的", "哪些人",
		"哪个人做的", "哪个人", "报工人", "操作员", "人员",
		"到哪儿了", "到哪了", "到哪里了", "做到哪儿了", "做到哪了", "做到哪里了",
		"做到哪一步了", "做到哪个工序了", "完成了吗", "完工了吗",
	}
	for _, old := range replacements {
		cleaned = strings.ReplaceAll(cleaned, old, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "，。？！? !：:")
	return cleaned
}

func extractQueryWithDeepSeek(ctx context.Context, message string) (extractedQuery, error) {
	payload := map[string]interface{}{
		"model":       config.Cfg.AI.DeepSeekModel,
		"temperature": 0,
		"stream":      false,
		"response_format": map[string]string{
			"type": "json_object",
		},
		"messages": []map[string]string{
			{
				"role": "system",
				"content": `你负责从扫码报工系统用户问题中提取查询参数，只返回 JSON。
JSON 字段：
{
  "intent": "query_orders"、"query_order_progress"、"query_order_report_details"、"query_order_reporters" 或 "unknown",
  "internal_no": "内部单号，没有则空字符串",
  "external_no": "外部单号/客户单号，没有则空字符串",
  "part_name": "零件名称关键词，没有则空字符串",
  "confidence": 0到1
}
规则：
1. 用户问工单基础资料、数量、图纸、单号、状态，intent 用 query_orders。
2. 用户问进度、做到哪、完成没有、哪个工序，intent 用 query_order_progress。
3. 用户问报工明细、报工记录、图片、备注，intent 用 query_order_report_details。
4. 用户问谁做的、谁报工、哪些人参与、哪个员工做，intent 用 query_order_reporters。
5. "查一下大的齿轮到哪儿了" 应提取 part_name 为 "齿轮"。
6. 不要把“查一下、进度、到哪儿了、单子、工单、订单、现在、目前、谁做的”等口语词放进 part_name。
7. 不要编造不存在的编号。`,
			},
			{
				"role":    "user",
				"content": message,
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deepSeekChatCompletionsURL(), bytes.NewReader(body))
	if err != nil {
		return extractedQuery{}, err
	}
	req.Header.Set("Authorization", "Bearer "+config.Cfg.AI.DeepSeekAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return extractedQuery{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return extractedQuery{}, fmt.Errorf("DeepSeek 提取参数失败：%s %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var completion deepSeekCompletion
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return extractedQuery{}, err
	}
	if len(completion.Choices) == 0 {
		return extractedQuery{}, fmt.Errorf("DeepSeek 未返回提取结果")
	}

	content := stripJSONFence(completion.Choices[0].Message.Content)
	var extracted extractedQuery
	if err := json.Unmarshal([]byte(content), &extracted); err != nil {
		return extractedQuery{}, err
	}
	return extracted, nil
}

func stripJSONFence(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func streamDeepSeekAnswer(c *gin.Context, message, toolName string, toolResult interface{}) (string, error) {
	contextBytes, _ := json.MarshalIndent(toolResult, "", "  ")
	payload := map[string]interface{}{
		"model":  config.Cfg.AI.DeepSeekModel,
		"stream": true,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "你是扫码报工系统的生产查询助手。你必须只根据 MCP 工具返回的数据回答，不要编造。回答要简洁。查进度时说明工单、零件、总进度、当前工序和各工序数量；查谁做的时说明报工人、参与工序、完成/接收/报废数量和最近报工时间；查明细时按时间列出关键明细。如查询结果为空，直接说明未找到。",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("用户问题：%s\n\nMCP工具 %s 返回：\n%s", message, toolName, string(contextBytes)),
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, deepSeekChatCompletionsURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+config.Cfg.AI.DeepSeekAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek 返回错误：%s %s", resp.Status, strings.TrimSpace(string(raw)))
	}

	var answer strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk deepSeekChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				answer.WriteString(choice.Delta.Content)
				writeSSE(c.Writer, "delta", choice.Delta.Content)
			}
		}
	}
	return answer.String(), scanner.Err()
}

func deepSeekChatCompletionsURL() string {
	baseURL := strings.TrimRight(config.Cfg.AI.DeepSeekBaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return baseURL + "/chat/completions"
}

func streamFallbackAnswer(c *gin.Context, message, toolName string, toolResult interface{}) string {
	var answer string
	switch toolName {
	case "query_orders":
		answer = fallbackOrdersText(toolResult)
	case "query_order_report_details":
		answer = fallbackReportDetailsText(toolResult)
	case "query_order_reporters":
		answer = fallbackReportersText(toolResult)
	default:
		answer = fallbackProgressText(toolResult)
	}
	for _, part := range splitRunes(answer, 32) {
		writeSSE(c.Writer, "delta", part)
	}
	writeSSE(c.Writer, "done", "")
	_ = message
	return answer
}

func fallbackProgressText(toolResult interface{}) string {
	raw, _ := json.Marshal(toolResult)
	var results []mcpTools.OrderProgressResult
	_ = json.Unmarshal(raw, &results)
	if len(results) == 0 {
		return "没有找到相关工单。"
	}

	var b strings.Builder
	b.WriteString("已通过 MCP 查询到以下进度：\n")
	for _, item := range results {
		order := item.Order
		b.WriteString(fmt.Sprintf("\n%s，外部单号 %s，零件 %s，数量 %d，已完成 %d，状态 %s。\n",
			order.InternalNo, emptyDash(order.ExternalNo), emptyDash(order.PartName), order.TotalQty, order.TotalCompleted, order.Status))
		if len(item.Progress) == 0 {
			b.WriteString("暂无工序进度。\n")
			continue
		}
		for _, p := range item.Progress {
			b.WriteString(fmt.Sprintf("- %v：状态 %v，接收 %v，完成 %v，报废 %v，进度 %v%%。\n",
				p["display_name"], p["status"], zeroValue(p["total_received"]), zeroValue(p["total_completed"]), zeroValue(p["total_scrap"]), zeroValue(p["progress_pct"])))
		}
	}
	b.WriteString("\n提示：当前未配置 DeepSeek API Key，以上为后端 MCP 查询结果摘要。")
	return b.String()
}

func fallbackOrdersText(toolResult interface{}) string {
	raw, _ := json.Marshal(toolResult)
	var orders []struct {
		InternalNo     string  `json:"internal_no"`
		ExternalNo     string  `json:"external_no"`
		PartName       string  `json:"part_name"`
		DrawingNo      string  `json:"drawing_no"`
		TotalQty       int     `json:"total_qty"`
		TotalCompleted int     `json:"total_completed"`
		Status         string  `json:"status"`
		UnitPrice      float64 `json:"unit_price"`
		TotalAmount    float64 `json:"total_amount"`
	}
	_ = json.Unmarshal(raw, &orders)
	if len(orders) == 0 {
		return "没有找到相关工单。"
	}

	var b strings.Builder
	b.WriteString("已通过 MCP 查询到工单信息：\n")
	for _, order := range orders {
		b.WriteString(fmt.Sprintf("\n%s，外部单号 %s，零件 %s，图纸编号 %s，数量 %d，已完成 %d，状态 %s，金额 %.2f。\n",
			order.InternalNo, emptyDash(order.ExternalNo), emptyDash(order.PartName), emptyDash(order.DrawingNo), order.TotalQty, order.TotalCompleted, order.Status, order.TotalAmount))
	}
	b.WriteString("\n提示：当前未配置 DeepSeek API Key，以上为后端 MCP 查询结果摘要。")
	return b.String()
}

func fallbackReportDetailsText(toolResult interface{}) string {
	raw, _ := json.Marshal(toolResult)
	var results []mcpTools.OrderReportDetailsResult
	_ = json.Unmarshal(raw, &results)
	if len(results) == 0 {
		return "没有找到相关报工明细。"
	}

	var b strings.Builder
	b.WriteString("已通过 MCP 查询到报工明细：\n")
	for _, item := range results {
		b.WriteString(fmt.Sprintf("\n%s，零件 %s：\n", item.Order.InternalNo, emptyDash(item.Order.PartName)))
		if len(item.Details) == 0 {
			b.WriteString("暂无报工明细。\n")
			continue
		}
		for _, detail := range item.Details {
			b.WriteString(fmt.Sprintf("- %s，%s，接收 %d，完成 %d，报废 %d，时间 %s。\n",
				emptyDash(detail.ProcessName), emptyDash(detail.UserName), detail.ReceivedQty, detail.CompletedQty, detail.ScrapQty, detail.ReportedAt.Format("2006-01-02 15:04")))
		}
	}
	b.WriteString("\n提示：当前未配置 DeepSeek API Key，以上为后端 MCP 查询结果摘要。")
	return b.String()
}

func fallbackReportersText(toolResult interface{}) string {
	raw, _ := json.Marshal(toolResult)
	var results []mcpTools.OrderReportersResult
	_ = json.Unmarshal(raw, &results)
	if len(results) == 0 {
		return "没有找到相关报工人。"
	}

	var b strings.Builder
	b.WriteString("已通过 MCP 查询到参与人员：\n")
	for _, item := range results {
		b.WriteString(fmt.Sprintf("\n%s，零件 %s：\n", item.Order.InternalNo, emptyDash(item.Order.PartName)))
		if len(item.Reporters) == 0 {
			b.WriteString("暂无报工人员记录。\n")
			continue
		}
		for _, reporter := range item.Reporters {
			b.WriteString(fmt.Sprintf("- %s，工序 %s，报工 %d 次，接收 %d，完成 %d，报废 %d，最近 %s。\n",
				emptyDash(reporter.UserName), strings.Join(reporter.ProcessNames, "/"), reporter.ReportCount, reporter.TotalReceived, reporter.TotalCompleted, reporter.TotalScrap, reporter.LastReportedAt.Format("2006-01-02 15:04")))
		}
	}
	b.WriteString("\n提示：当前未配置 DeepSeek API Key，以上为后端 MCP 查询结果摘要。")
	return b.String()
}

func streamError(c *gin.Context, message string) {
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	writeSSE(c.Writer, "error", message)
	writeSSE(c.Writer, "done", "")
}

func writeSSE(w gin.ResponseWriter, event, data string) {
	payload, _ := json.Marshal(data)
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload)
	w.Flush()
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func zeroValue(v interface{}) interface{} {
	if v == nil {
		return 0
	}
	return v
}

func splitRunes(s string, size int) []string {
	if size <= 0 {
		return []string{s}
	}
	runes := []rune(s)
	var parts []string
	for len(runes) > 0 {
		n := size
		if len(runes) < n {
			n = len(runes)
		}
		parts = append(parts, string(runes[:n]))
		runes = runes[n:]
	}
	return parts
}
