package handlers

import (
	"fmt"
	"net/http"
	"parking-system/config"
	"parking-system/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SpaceHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewSpaceHandler(db *gorm.DB, cfg *config.Config) *SpaceHandler {
	return &SpaceHandler{db: db, cfg: cfg}
}

func (h *SpaceHandler) Status(c *gin.Context) {
	var total, occupied, free int64
	h.db.Model(&models.ParkingSpace{}).Count(&total)
	h.db.Model(&models.ParkingSpace{}).Where("status = ?", "occupied").Count(&occupied)
	h.db.Model(&models.ParkingSpace{}).Where("status = ?", "free").Count(&free)
	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"occupied": occupied,
		"free":     free,
	})
}

func (h *SpaceHandler) List(c *gin.Context) {
	status := c.Query("status")
	query := h.db.Model(&models.ParkingSpace{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var spaces []models.ParkingSpace
	query.Order("space_no ASC").Find(&spaces)
	c.JSON(http.StatusOK, gin.H{"list": spaces, "count": len(spaces)})
}

type AddSpacesReq struct {
	Count int `json:"count" binding:"required,min=1"`
}

func (h *SpaceHandler) AddSpaces(c *gin.Context) {
	var req AddSpacesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var maxID int64
	h.db.Model(&models.ParkingSpace{}).Count(&maxID)
	spaces := make([]models.ParkingSpace, 0, req.Count)
	for i := 1; i <= req.Count; i++ {
		spaces = append(spaces, models.ParkingSpace{
			SpaceNo: fmt.Sprintf("B%03d", maxID+int64(i)),
			Status:  "free",
		})
	}
	h.db.Create(&spaces)
	c.JSON(http.StatusOK, gin.H{"message": "成功添加车位", "count": req.Count})
}
