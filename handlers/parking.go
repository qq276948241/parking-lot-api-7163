package handlers

import (
	"math"
	"net/http"
	"parking-system/config"
	"parking-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ParkingHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewParkingHandler(db *gorm.DB, cfg *config.Config) *ParkingHandler {
	return &ParkingHandler{db: db, cfg: cfg}
}

type EntryReq struct {
	PlateNo string `json:"plate_no" binding:"required"`
}

type ExitReq struct {
	PlateNo string `json:"plate_no" binding:"required"`
	PayType string `json:"pay_type"`
}

func (h *ParkingHandler) Entry(c *gin.Context) {
	var req EntryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	var existing models.ParkingRecord
	h.db.Where("plate_no = ? AND status = ?", req.PlateNo, "parking").First(&existing)
	if existing.ID > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该车辆已在场内"})
		return
	}

	var space models.ParkingSpace
	h.db.Where("status = ?", "free").First(&space)
	if space.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有空闲车位"})
		return
	}

	var monthlyCard models.MonthlyCard
	isMonthly := false
	now := time.Now()
	h.db.Where("plate_no = ? AND status = ? AND start_date <= ? AND end_date >= ?",
		req.PlateNo, "active", now, now).First(&monthlyCard)
	if monthlyCard.ID > 0 {
		isMonthly = true
	}

	operator, _ := c.Get("username")
	record := models.ParkingRecord{
		PlateNo:   req.PlateNo,
		SpaceNo:   space.SpaceNo,
		EntryTime: now,
		IsMonthly: isMonthly,
		Operator:  operator.(string),
		Status:    "parking",
	}
	h.db.Create(&record)

	space.Status = "occupied"
	space.PlateNo = req.PlateNo
	space.RecordID = &record.ID
	h.db.Save(&space)

	c.JSON(http.StatusOK, gin.H{
		"message":   "进场成功",
		"record":    record,
		"is_monthly": isMonthly,
	})
}

func (h *ParkingHandler) Exit(c *gin.Context) {
	var req ExitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if req.PayType == "" {
		req.PayType = "cash"
	}

	var record models.ParkingRecord
	h.db.Where("plate_no = ? AND status = ?", req.PlateNo, "parking").First(&record)
	if record.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到该车辆的进场记录"})
		return
	}

	now := time.Now()
	record.ExitTime = &now
	duration := int(now.Sub(record.EntryTime).Minutes())
	record.Duration = duration

	fee := 0.0
	if !record.IsMonthly {
		fee = calculateFee(duration, h.cfg.Parking.HourlyRate, h.cfg.Parking.MaxDailyRate, h.cfg.Parking.FreeMinutes)
	}
	record.Fee = fee
	record.Status = "exited"
	exitOperator, _ := c.Get("username")
	record.ExitOperator = exitOperator.(string)
	h.db.Save(&record)

	var space models.ParkingSpace
	h.db.Where("space_no = ?", record.SpaceNo).First(&space)
	if space.ID > 0 {
		space.Status = "free"
		space.PlateNo = ""
		space.RecordID = nil
		h.db.Save(&space)
	}

	if fee > 0 {
		bill := models.Bill{
			RecordID:  record.ID,
			PlateNo:   record.PlateNo,
			Amount:    fee,
			PayType:   req.PayType,
			PayTime:   now,
			IsMonthly: false,
			Operator:  exitOperator.(string),
		}
		h.db.Create(&bill)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "出场成功",
		"record":     record,
		"fee":        fee,
		"is_monthly": record.IsMonthly,
	})
}

func calculateFee(minutes, hourlyRate, maxDailyRate, freeMinutes int) float64 {
	if minutes <= freeMinutes {
		return 0
	}
	effectiveMin := minutes - freeMinutes
	hours := int(math.Ceil(float64(effectiveMin) / 60))
	days := hours / 24
	remainHours := hours % 24
	fee := float64(days*maxDailyRate + remainHours*hourlyRate)
	if hours > 0 && fee > float64(days+1)*float64(maxDailyRate) {
		fee = float64(days+1) * float64(maxDailyRate)
	}
	return fee
}

func (h *ParkingHandler) ListRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	plateNo := c.Query("plate_no")
	status := c.Query("status")

	query := h.db.Model(&models.ParkingRecord{})
	if plateNo != "" {
		query = query.Where("plate_no LIKE ?", "%"+plateNo+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var records []models.ParkingRecord
	query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&records)

	c.JSON(http.StatusOK, gin.H{"total": total, "list": records})
}

func (h *ParkingHandler) GetRecord(c *gin.Context) {
	id := c.Param("id")
	var record models.ParkingRecord
	if err := h.db.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}
	c.JSON(http.StatusOK, record)
}
