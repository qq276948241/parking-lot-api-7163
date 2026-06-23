package models

import (
	"fmt"
	"parking-system/config"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	RealName  string    `gorm:"size:50" json:"real_name"`
	Role      string    `gorm:"size:20;default:operator" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ParkingSpace struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SpaceNo   string    `gorm:"uniqueIndex;size:20;not null" json:"space_no"`
	Status    string    `gorm:"size:20;default:free" json:"status"`
	PlateNo   string    `gorm:"size:20" json:"plate_no,omitempty"`
	RecordID  *uint     `gorm:"index" json:"record_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ParkingRecord struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	PlateNo      string     `gorm:"index;size:20;not null" json:"plate_no"`
	SpaceNo      string     `gorm:"size:20" json:"space_no,omitempty"`
	EntryTime    time.Time  `gorm:"not null" json:"entry_time"`
	ExitTime     *time.Time `json:"exit_time,omitempty"`
	Duration     int        `json:"duration,omitempty"`
	Fee          float64    `json:"fee,omitempty"`
	IsMonthly    bool       `gorm:"default:false" json:"is_monthly"`
	Operator     string     `gorm:"size:50" json:"operator,omitempty"`
	ExitOperator string     `gorm:"size:50" json:"exit_operator,omitempty"`
	Status       string     `gorm:"size:20;default:parking" json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type MonthlyCard struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PlateNo   string    `gorm:"uniqueIndex;size:20;not null" json:"plate_no"`
	OwnerName string    `gorm:"size:50" json:"owner_name"`
	Phone     string    `gorm:"size:20" json:"phone"`
	StartDate time.Time `gorm:"not null" json:"start_date"`
	EndDate   time.Time `gorm:"not null" json:"end_date"`
	Price     float64   `json:"price"`
	Status    string    `gorm:"size:20;default:active" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Bill struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	RecordID  uint      `gorm:"index;not null" json:"record_id"`
	PlateNo   string    `gorm:"index;size:20;not null" json:"plate_no"`
	Amount    float64   `json:"amount"`
	PayType   string    `gorm:"size:20;default:cash" json:"pay_type"`
	PayTime   time.Time `gorm:"not null" json:"pay_time"`
	IsMonthly bool      `gorm:"default:false" json:"is_monthly"`
	Operator  string    `gorm:"size:50" json:"operator,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&User{}, &ParkingSpace{}, &ParkingRecord{}, &MonthlyCard{}, &Bill{})
	if err != nil {
		return nil, err
	}

	var count int64
	db.Model(&User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Create(&User{
			Username: "admin",
			Password: string(hashedPwd),
			RealName: "超级管理员",
			Role:     "admin",
		})
	}

	return db, nil
}

func InitParkingSpaces(db *gorm.DB, total int) {
	var count int64
	db.Model(&ParkingSpace{}).Count(&count)
	if count > 0 {
		return
	}
	spaces := make([]ParkingSpace, 0, total)
	for i := 1; i <= total; i++ {
		spaces = append(spaces, ParkingSpace{
			SpaceNo: fmt.Sprintf("A%03d", i),
			Status:  "free",
		})
	}
	db.Create(&spaces)
}
