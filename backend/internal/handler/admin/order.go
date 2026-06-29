package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sui/scan-report/internal/database"
	"github.com/sui/scan-report/internal/model"
	"github.com/sui/scan-report/internal/repository"
	"github.com/sui/scan-report/internal/service"
	"gorm.io/gorm"
)

// ListOrders GET /api/admin/orders
func ListOrders(c *gin.Context) {
	var orders []model.Order
	query := database.DB.Preload("Creator")

	if t := c.Query("order_type"); t != "" {
		query = query.Where("order_type = ?", t)
	}
	if s := c.Query("status"); s != "" {
		query = query.Where("status = ?", s)
	}
	if kw := c.Query("keyword"); kw != "" {
		query = query.Where("internal_no LIKE ? OR external_no LIKE ? OR drawing_no LIKE ? OR part_name LIKE ?",
			"%"+kw+"%", "%"+kw+"%", "%"+kw+"%", "%"+kw+"%")
	}
	if internalNo := c.Query("internal_no"); internalNo != "" {
		query = query.Where("internal_no LIKE ?", "%"+internalNo+"%")
	}
	if externalNo := c.Query("external_no"); externalNo != "" {
		query = query.Where("external_no LIKE ?", "%"+externalNo+"%")
	}
	if partName := c.Query("part_name"); partName != "" {
		query = query.Where("part_name LIKE ?", "%"+partName+"%")
	}

	var total int64
	query.Model(&model.Order{}).Count(&total)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	query.Offset((page - 1) * pageSize).Limit(pageSize).
		Order("created_at DESC").Find(&orders)

	c.JSON(http.StatusOK, gin.H{"data": orders, "total": total})
}

// CreateOrder POST /api/admin/orders
func CreateOrder(c *gin.Context) {
	var order model.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order.CreatedBy = c.GetUint("user_id")
	order.OrderType = model.OrderTypeNormal
	order.Status = model.OrderStatusDraft

	var created model.Order
	var err error
	for i := 0; i < 3; i++ {
		err = database.DB.Transaction(func(tx *gorm.DB) error {
			internalNo, genErr := repository.GenerateInternalNo(tx, time.Now())
			if genErr != nil {
				return genErr
			}
			order.InternalNo = internalNo
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			created = order
			return nil
		})
		if err == nil {
			c.JSON(http.StatusCreated, gin.H{"data": created})
			return
		}
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// GetOrder GET /api/admin/orders/:id
func GetOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var order model.Order
	if err := database.DB.
		Preload("Processes.Summary").
		Preload("Processes.Process").
		Preload("ScrapOrders").
		First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

// UpdateOrder PUT /api/admin/orders/:id
func UpdateOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var order model.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	internalNo := order.InternalNo
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order.InternalNo = internalNo
	database.DB.Save(&order)
	c.JSON(http.StatusOK, gin.H{"data": order})
}

// DeleteOrder DELETE /api/admin/orders/:id
func DeleteOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var order model.Order
	if err := database.DB.First(&order, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if order.Status != model.OrderStatusDraft {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有草稿状态的工单可以删除"})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", order.ID).Delete(&model.OrderProcess{}).Error; err != nil {
			return err
		}
		return tx.Delete(&order).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// CreateScrapOrder POST /api/admin/orders/:id/scrap
func CreateScrapOrder(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID := c.GetUint("user_id")
	order, err := service.CreateScrapOrder(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": order})
}

// GetScrapOrders GET /api/admin/orders/:id/scrap-orders
func GetScrapOrders(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var orders []model.Order
	database.DB.Where("parent_id = ? AND order_type = ?", id, model.OrderTypeScrap).Find(&orders)
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

// AddOrderProcess POST /api/admin/orders/:id/processes
func AddOrderProcess(c *gin.Context) {
	orderID, _ := strconv.Atoi(c.Param("id"))
	var op model.OrderProcess
	if err := c.ShouldBindJSON(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	op.OrderID = uint(orderID)
	op.Status = model.ProcessStatusPending
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&op).Error; err != nil {
			return err
		}
		return tx.Model(&model.Order{}).
			Where("id = ? AND status = ?", orderID, model.OrderStatusDraft).
			Update("status", model.OrderStatusReady).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": op})
}

// UpdateOrderProcess PUT /api/admin/orders/:id/processes/:pid
func UpdateOrderProcess(c *gin.Context) {
	pid, _ := strconv.Atoi(c.Param("pid"))
	var op model.OrderProcess
	if err := database.DB.First(&op, pid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := c.ShouldBindJSON(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	database.DB.Save(&op)
	c.JSON(http.StatusOK, gin.H{"data": op})
}

// GetReportProgress GET /api/admin/orders/:id/report-progress
func GetReportProgress(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	progress, err := service.GetOrderProgress(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": progress})
}

// GetOrderProcessReportDetails GET /api/admin/order-processes/:id/report-details
func GetOrderProcessReportDetails(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var details []model.WorkReportDetail
	if err := database.DB.
		Preload("User").
		Where("order_process_id = ?", id).
		Order("reported_at DESC").
		Find(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": details})
}
