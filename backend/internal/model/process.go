package model

// Process 工序模板（只存名称，不存工时）
type Process struct {
	BaseModel
	Name         string      `gorm:"size:100;not null" json:"name"`
	DepartmentID *uint       `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
}

// ProcessAlias 流程模板（一组有序工序快照，用于一键导入）
type ProcessAlias struct {
	BaseModel
	AliasName    string             `gorm:"size:100;not null" json:"alias_name"`
	DepartmentID *uint              `json:"department_id"`
	Department   *Department        `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	Note         string             `gorm:"size:255" json:"note"`
	Items        []ProcessAliasItem `gorm:"foreignKey:AliasID" json:"items,omitempty"`
}

// ProcessAliasItem 流程模板明细（只含工序名称和排序，不含工时）
type ProcessAliasItem struct {
	BaseModel
	AliasID     uint    `gorm:"not null;index" json:"alias_id"`
	ProcessID   uint    `gorm:"not null" json:"process_id"`
	Process     Process `gorm:"foreignKey:ProcessID" json:"process,omitempty"`
	DisplayName string  `gorm:"size:100" json:"display_name"` // 工序别名
	Sort        int     `gorm:"not null;default:0" json:"sort"`
}
