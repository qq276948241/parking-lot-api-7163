package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"parking-system/config"
	"parking-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BillHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewBillHandler(db *gorm.DB, cfg *config.Config) *BillHandler {
	return &BillHandler{db: db, cfg: cfg}
}

func (h *BillHandler) DailyStats(c *gin.Context) {
	date := c.Query("date")
	var startTime, endTime time.Time
	var err error
	if date == "" {
		startTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
		endTime = startTime.AddDate(0, 0, 1)
	} else {
		startTime, err = time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日期格式错误，应为 YYYY-MM-DD"})
			return
		}
		endTime = startTime.AddDate(0, 0, 1)
	}

	var totalAmount float64
	var totalCount int64
	h.db.Model(&models.Bill{}).Where("pay_time >= ? AND pay_time < ?", startTime, endTime).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	h.db.Model(&models.Bill{}).Where("pay_time >= ? AND pay_time < ?", startTime, endTime).Count(&totalCount)

	type HourlyStat struct {
		Hour   int     `json:"hour"`
		Amount float64 `json:"amount"`
		Count  int64   `json:"count"`
	}
	var hourly []HourlyStat
	h.db.Model(&models.Bill{}).
		Select("HOUR(pay_time) as hour, COALESCE(SUM(amount), 0) as amount, COUNT(*) as count").
		Where("pay_time >= ? AND pay_time < ?", startTime, endTime).
		Group("HOUR(pay_time)").
		Order("hour ASC").
		Scan(&hourly)

	c.JSON(http.StatusOK, gin.H{
		"date":         startTime.Format("2006-01-02"),
		"total_amount": totalAmount,
		"total_count":  totalCount,
		"hourly":       hourly,
	})
}

func (h *BillHandler) MonthlyStats(c *gin.Context) {
	month := c.Query("month")
	var startTime, endTime time.Time
	var err error
	if month == "" {
		now := time.Now()
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		endTime = startTime.AddDate(0, 1, 0)
	} else {
		startTime, err = time.ParseInLocation("2006-01", month, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "月份格式错误，应为 YYYY-MM"})
			return
		}
		endTime = startTime.AddDate(0, 1, 0)
	}

	var totalAmount float64
	var totalCount int64
	h.db.Model(&models.Bill{}).Where("pay_time >= ? AND pay_time < ?", startTime, endTime).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalAmount)
	h.db.Model(&models.Bill{}).Where("pay_time >= ? AND pay_time < ?", startTime, endTime).Count(&totalCount)

	type DailyStat struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
		Count  int64   `json:"count"`
	}
	var daily []DailyStat
	h.db.Model(&models.Bill{}).
		Select("DATE(pay_time) as date, COALESCE(SUM(amount), 0) as amount, COUNT(*) as count").
		Where("pay_time >= ? AND pay_time < ?", startTime, endTime).
		Group("DATE(pay_time)").
		Order("date ASC").
		Scan(&daily)

	c.JSON(http.StatusOK, gin.H{
		"month":        startTime.Format("2006-01"),
		"total_amount": totalAmount,
		"total_count":  totalCount,
		"daily":        daily,
	})
}

func (h *BillHandler) ListBills(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	plateNo := c.Query("plate_no")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := h.db.Model(&models.Bill{})
	if plateNo != "" {
		query = query.Where("plate_no LIKE ?", "%"+plateNo+"%")
	}
	if startDate != "" {
		query = query.Where("pay_time >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("pay_time < ?", endDate+" 23:59:59")
	}

	var total int64
	query.Count(&total)

	var bills []models.Bill
	query.Order("pay_time DESC").Offset((page - 1) * size).Limit(size).Find(&bills)

	c.JSON(http.StatusOK, gin.H{"total": total, "list": bills})
}

func (h *BillHandler) Export(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	query := h.db.Model(&models.Bill{})
	if startDate != "" {
		query = query.Where("pay_time >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("pay_time < ?", endDate+" 23:59:59")
	}

	var bills []models.Bill
	query.Order("pay_time DESC").Find(&bills)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=bills_%s.csv", time.Now().Format("20060102_150405")))
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"账单ID", "停车记录ID", "车牌号", "金额(元)", "支付方式", "支付时间", "是否月卡", "操作员"})
	for _, b := range bills {
		w.Write([]string{
			strconv.Itoa(int(b.ID)),
			strconv.Itoa(int(b.RecordID)),
			b.PlateNo,
			fmt.Sprintf("%.2f", b.Amount),
			b.PayType,
			b.PayTime.Format("2006-01-02 15:04:05"),
			map[bool]string{true: "是", false: "否"}[b.IsMonthly],
			b.Operator,
		})
	}
	w.Flush()
	c.Status(http.StatusOK)
}
