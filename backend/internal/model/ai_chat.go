package model

// AIChatSession AI 对话会话
type AIChatSession struct {
	BaseModel
	UserID        uint   `gorm:"not null;index" json:"user_id"`
	User          *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Title         string `gorm:"size:100" json:"title"`
	LastQueryArgs string `gorm:"type:jsonb;default:'{}'" json:"last_query_args"`
}

// AIChatMessage AI 对话消息
type AIChatMessage struct {
	BaseModel
	SessionID         uint           `gorm:"not null;index" json:"session_id"`
	Session           *AIChatSession `gorm:"foreignKey:SessionID" json:"session,omitempty"`
	Role              string         `gorm:"size:20;not null" json:"role"`
	Content           string         `gorm:"type:text" json:"content"`
	ToolName          string         `gorm:"size:100" json:"tool_name"`
	ToolArgs          string         `gorm:"type:jsonb;default:'{}'" json:"tool_args"`
	ToolResultSummary string         `gorm:"type:text" json:"tool_result_summary"`
}
