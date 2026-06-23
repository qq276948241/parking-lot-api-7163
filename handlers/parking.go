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

type FeeBreakdown struct {
	DayDuration   int     `json:"day_duration"`
	NightDuration int     `json:"night_duration"`
	DayFee        float64 `json:"day_fee"`
	NightFee      float64 `json:"night_fee"`
	TotalFee      float64 `json:"total_fee"`
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
		"message":    "进场成功",
		"record":     record,
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

	breakdown := FeeBreakdown{}
	if !record.IsMonthly {
		breakdown = calculateFeeByPeriod(
			record.EntryTime, now,
			h.cfg.Parking.DaytimeRate, h.cfg.Parking.NighttimeRate,
			h.cfg.Parking.DaytimeStart, h.cfg.Parking.DaytimeEnd,
			h.cfg.Parking.MaxDailyRate, h.cfg.Parking.FreeMinutes,
		)
		record.DayDuration = breakdown.DayDuration
		record.NightDuration = breakdown.NightDuration
		record.DayFee = breakdown.DayFee
		record.NightFee = breakdown.NightFee
		record.Fee = breakdown.TotalFee
	}
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

	if record.Fee > 0 {
		bill := models.Bill{
			RecordID:  record.ID,
			PlateNo:   record.PlateNo,
			Amount:    record.Fee,
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
		"fee":        record.Fee,
		"breakdown":  breakdown,
		"is_monthly": record.IsMonthly,
	})
}

func calculateFeeByPeriod(entry, exit time.Time, daytimeRate, nighttimeRate, daytimeStart, daytimeEnd, maxDailyRate, freeMinutes int) FeeBreakdown {
	totalMin := int(exit.Sub(entry).Minutes())
	if totalMin <= 0 {
		return FeeBreakdown{}
	}
	effectiveMin := totalMin - freeMinutes
	if effectiveMin <= 0 {
		return FeeBreakdown{}
	}

	dayMin := 0
	nightMin := 0

	cursor := entry
	effectiveStart := entry.Add(time.Duration(freeMinutes) * time.Minute)
	if effectiveStart.After(exit) {
		return FeeBreakdown{}
	}
	cursor = effectiveStart

	for cursor.Before(exit) {
		next := cursor.Add(time.Minute)
		if next.After(exit) {
			next = exit
		}
		hour := cursor.Hour()
		if hour >= daytimeStart && hour < daytimeEnd {
			dayMin += int(next.Sub(cursor).Minutes())
		} else {
			nightMin += int(next.Sub(cursor).Minutes())
		}
		cursor = next
	}

	dayHours := 0
	if dayMin > 0 {
		dayHours = int(math.Ceil(float64(dayMin) / 60))
	}
	nightHours := 0
	if nightMin > 0 {
		nightHours = int(math.Ceil(float64(nightMin) / 60))
	}

	dayFee := float64(dayHours * daytimeRate)
	nightFee := float64(nightHours * nighttimeRate)

	days := effectiveMin / (24 * 60)
	remainMin := effectiveMin % (24 * 60)
	totalFee := dayFee + nightFee

	if days > 0 {
		dailyFee := float64(maxDailyRate)
		remainHours := int(math.Ceil(float64(remainMin) / 60))
		_ = remainHours
		totalFee = float64(days)*dailyFee + math.Min(dayFee+nightFee, dailyFee)
	} else {
		if totalFee > float64(maxDailyRate) {
			totalFee = float64(maxDailyRate)
		}
	}

	return FeeBreakdown{
		DayDuration:   dayMin,
		NightDuration: nightMin,
		DayFee:        dayFee,
		NightFee:      nightFee,
		TotalFee:      totalFee,
	}
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
