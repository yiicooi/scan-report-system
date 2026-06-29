package app

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"gorm.io/gorm"
)

type aiChatHistoryMessage struct {
	ID        uint   `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// AIChatMessages GET /api/app/ai/chat/messages
func AIChatMessages(c *gin.Context) {
	userID := c.GetUint("user_id")
	session, err := getOrCreateAIChatSession(userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var messages []model.AIChatMessage
	if err := database.DB.
		Where("session_id = ?", session.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&messages).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	result := make([]aiChatHistoryMessage, 0, len(messages))
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		result = append(result, aiChatHistoryMessage{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(200, gin.H{"data": result})
}

func getOrCreateAIChatSession(userID uint) (*model.AIChatSession, error) {
	var session model.AIChatSession
	err := database.DB.
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		First(&session).Error
	if err == nil {
		return &session, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	session = model.AIChatSession{
		UserID:        userID,
		Title:         "AI 查询",
		LastQueryArgs: "{}",
	}
	if err := database.DB.Create(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func createAIChatMessage(sessionID uint, role, content, toolName string, toolArgs map[string]interface{}, toolResult interface{}) {
	content = strings.TrimSpace(content)
	if sessionID == 0 || role == "" || content == "" {
		return
	}
	database.DB.Create(&model.AIChatMessage{
		SessionID:         sessionID,
		Role:              role,
		Content:           content,
		ToolName:          toolName,
		ToolArgs:          marshalJSONText(toolArgs),
		ToolResultSummary: truncateText(marshalJSONText(toolResult), 4000),
	})
}

func updateAIChatSessionQueryArgs(sessionID uint, args map[string]interface{}) {
	if sessionID == 0 || !hasQueryTarget(args) {
		return
	}
	database.DB.Model(&model.AIChatSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"last_query_args": marshalJSONText(normalizeQueryArgs(args)),
		})
}

func loadLastQueryArgs(userID uint) map[string]interface{} {
	if userID == 0 {
		return nil
	}
	var session model.AIChatSession
	if err := database.DB.
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		First(&session).Error; err != nil {
		return nil
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(session.LastQueryArgs), &args); err != nil {
		return nil
	}
	return args
}

func normalizeQueryArgs(args map[string]interface{}) map[string]interface{} {
	normalized := map[string]interface{}{"limit": 5}
	for _, key := range []string{"internal_no", "external_no", "part_name"} {
		if v := strings.TrimSpace(stringValue(args[key])); v != "" {
			normalized[key] = v
		}
	}
	return normalized
}

func marshalJSONText(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func truncateText(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
