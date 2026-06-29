package app

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/service"
)

// Scan GET /api/app/scan?internal_no=xxx
func Scan(c *gin.Context) {
	internalNo := c.Query("internal_no")
	if internalNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "internal_no required"})
		return
	}
	userID := c.GetUint("user_id")
	order, err := service.ScanOrderForUser(internalNo, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

// SubmitReport POST /api/app/report
func SubmitReport(c *gin.Context) {
	var input struct {
		OrderProcessID uint              `json:"order_process_id" binding:"required"`
		ReceivedQty    int               `json:"received_qty"`
		CompletedQty   int               `json:"completed_qty"`
		ScrapQty       int               `json:"scrap_qty"`
		ReceiveImages  model.StringSlice `json:"receive_images"`
		CompleteImages model.StringSlice `json:"complete_images"`
		ScrapImages    model.StringSlice `json:"scrap_images"`
		Note           string            `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	detail, err := service.SubmitReport(service.ReportInput{
		OrderProcessID: input.OrderProcessID,
		UserID:         userID,
		ReceivedQty:    input.ReceivedQty,
		CompletedQty:   input.CompletedQty,
		ScrapQty:       input.ScrapQty,
		ReceiveImages:  input.ReceiveImages,
		CompleteImages: input.CompleteImages,
		ScrapImages:    input.ScrapImages,
		Note:           input.Note,
	})
	if err != nil {
		status := http.StatusInternalServerError
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			status = http.StatusUnprocessableEntity
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": detail})
}

// ReportHistory GET /api/app/report/history
func ReportHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	_ = userID
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
}

// OrderProgress GET /api/app/orders/:id/progress
func OrderProgress(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	progress, err := service.GetOrderProgress(uri.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": progress})
}
