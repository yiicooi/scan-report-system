package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/pkg/oss"
)

// ListProcesses GET /api/admin/processes
func ListProcesses(c *gin.Context) {
	var list []model.Process
	database.DB.Preload("Department").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// CreateProcess POST /api/admin/processes
func CreateProcess(c *gin.Context) {
	var p model.Process
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Create(&p)
	c.JSON(http.StatusCreated, gin.H{"data": p})
}

// UpdateProcess PUT /api/admin/processes/:id
func UpdateProcess(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var p model.Process
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Save(&p)
	c.JSON(http.StatusOK, gin.H{"data": p})
}

// DeleteProcess DELETE /api/admin/processes/:id
func DeleteProcess(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var count int64
	database.DB.Model(&model.OrderProcess{}).Where("process_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该工序已被工单引用，无法删除"})
		return
	}

	database.DB.Delete(&model.Process{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListProcessAliases GET /api/admin/process-aliases
func ListProcessAliases(c *gin.Context) {
	var list []model.ProcessAlias
	database.DB.Preload("Items.Process").Find(&list)
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// CreateProcessAlias POST /api/admin/process-aliases
func CreateProcessAlias(c *gin.Context) {
	var alias model.ProcessAlias
	if err := c.ShouldBindJSON(&alias); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Create(&alias).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": alias})
}

// GetProcessAlias GET /api/admin/process-aliases/:id
func GetProcessAlias(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var alias model.ProcessAlias
	if err := database.DB.Preload("Items.Process").First(&alias, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": alias})
}

// DeleteProcessAlias DELETE /api/admin/process-aliases/:id
func DeleteProcessAlias(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	database.DB.Delete(&model.ProcessAlias{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Presign POST /api/admin/oss/presign  — 生成预签名上传 URL
func Presign(c *gin.Context) {
	var input struct {
		Filename string `json:"filename" binding:"required"`
		Prefix   string `json:"prefix"` // e.g. drawings / reports
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.Prefix == "" {
		input.Prefix = "uploads"
	}
	objectKey := fmt.Sprintf("%s/%s/%d_%s",
		input.Prefix,
		time.Now().Format("2006/01/02"),
		time.Now().UnixMilli(),
		input.Filename,
	)
	url, err := oss.PresignPutURL(objectKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"upload_url": url,
		"object_key": objectKey,
		"access_url": oss.ObjectURL(objectKey),
	})
}
