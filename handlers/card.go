package handlers

import (
	"net/http"
	"parking-system/config"
	"parking-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CardHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewCardHandler(db *gorm.DB, cfg *config.Config) *CardHandler {
	return &CardHandler{db: db, cfg: cfg}
}

type CreateCardReq struct {
	PlateNo   string `json:"plate_no" binding:"required"`
	OwnerName string `json:"owner_name"`
	Phone     string `json:"phone"`
	Months    int    `json:"months" binding:"required,min=1"`
}

func (h *CardHandler) CreateCard(c *gin.Context) {
	var req CreateCardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var existing models.MonthlyCard
	h.db.Where("plate_no = ?", req.PlateNo).First(&existing)
	if existing.ID > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该车牌号已办理月卡"})
		return
	}

	now := time.Now()
	startDate := now
	endDate := now.AddDate(0, req.Months, 0)
	price := float64(req.Months) * float64(h.cfg.Parking.MonthlyCardPrice)

	card := models.MonthlyCard{
		PlateNo:   req.PlateNo,
		OwnerName: req.OwnerName,
		Phone:     req.Phone,
		StartDate: startDate,
		EndDate:   endDate,
		Price:     price,
		Status:    "active",
	}
	h.db.Create(&card)

	c.JSON(http.StatusOK, gin.H{
		"message": "开卡成功",
		"card":    card,
	})
}

type RenewCardReq struct {
	PlateNo string `json:"plate_no" binding:"required"`
	Months  int    `json:"months" binding:"required,min=1"`
}

func (h *CardHandler) RenewCard(c *gin.Context) {
	var req RenewCardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var card models.MonthlyCard
	h.db.Where("plate_no = ?", req.PlateNo).First(&card)
	if card.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到该月卡"})
		return
	}

	addPrice := float64(req.Months) * float64(h.cfg.Parking.MonthlyCardPrice)
	now := time.Now()
	var newStart time.Time
	if card.EndDate.After(now) {
		newStart = card.EndDate
	} else {
		newStart = now
	}
	card.EndDate = newStart.AddDate(0, req.Months, 0)
	card.Price += addPrice
	if card.Status == "expired" {
		card.Status = "active"
	}
	h.db.Save(&card)

	c.JSON(http.StatusOK, gin.H{
		"message":      "续费成功",
		"card":         card,
		"renew_price":  addPrice,
		"renew_months": req.Months,
	})
}

func (h *CardHandler) ListCards(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	plateNo := c.Query("plate_no")
	status := c.Query("status")

	now := time.Now()
	h.db.Model(&models.MonthlyCard{}).Where("end_date < ? AND status = ?", now, "active").Update("status", "expired")

	query := h.db.Model(&models.MonthlyCard{})
	if plateNo != "" {
		query = query.Where("plate_no LIKE ?", "%"+plateNo+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var cards []models.MonthlyCard
	query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&cards)

	c.JSON(http.StatusOK, gin.H{"total": total, "list": cards})
}

func (h *CardHandler) GetCard(c *gin.Context) {
	id := c.Param("id")
	var card models.MonthlyCard
	if err := h.db.First(&card, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "月卡不存在"})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CardHandler) GetCardByPlate(c *gin.Context) {
	plateNo := c.Param("plate")
	now := time.Now()
	var card models.MonthlyCard
	h.db.Where("plate_no = ? AND status = ? AND start_date <= ? AND end_date >= ?",
		plateNo, "active", now, now).First(&card)
	if card.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到有效的月卡"})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *CardHandler) DeleteCard(c *gin.Context) {
	id := c.Param("id")
	h.db.Delete(&models.MonthlyCard{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
