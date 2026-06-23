package services

import (
	"math"
	"parking-system/config"
	"time"
)

type FeeBreakdown struct {
	DayDuration   int     `json:"day_duration"`
	NightDuration int     `json:"night_duration"`
	DayFee        float64 `json:"day_fee"`
	NightFee      float64 `json:"night_fee"`
	TotalFee      float64 `json:"total_fee"`
}

type ParkingService struct {
	cfg *config.Config
}

func NewParkingService(cfg *config.Config) *ParkingService {
	return &ParkingService{cfg: cfg}
}

func (s *ParkingService) CalculateFee(entry, exit time.Time, isMonthly bool) FeeBreakdown {
	if isMonthly {
		return FeeBreakdown{}
	}
	return s.calculateFeeByPeriod(entry, exit)
}

func (s *ParkingService) calculateFeeByPeriod(entry, exit time.Time) FeeBreakdown {
	p := s.cfg.Parking
	totalMin := int(exit.Sub(entry).Minutes())
	if totalMin <= 0 {
		return FeeBreakdown{}
	}
	effectiveMin := totalMin - p.FreeMinutes
	if effectiveMin <= 0 {
		return FeeBreakdown{}
	}

	dayMin := 0
	nightMin := 0

	effectiveStart := entry.Add(time.Duration(p.FreeMinutes) * time.Minute)
	if effectiveStart.After(exit) {
		return FeeBreakdown{}
	}
	cursor := effectiveStart

	for cursor.Before(exit) {
		next := cursor.Add(time.Minute)
		if next.After(exit) {
			next = exit
		}
		hour := cursor.Hour()
		if hour >= p.DaytimeStart && hour < p.DaytimeEnd {
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

	dayFee := float64(dayHours * p.DaytimeRate)
	nightFee := float64(nightHours * p.NighttimeRate)

	days := effectiveMin / (24 * 60)
	totalFee := dayFee + nightFee

	if days > 0 {
		dailyFee := float64(p.MaxDailyRate)
		totalFee = float64(days)*dailyFee + math.Min(dayFee+nightFee, dailyFee)
	} else {
		if totalFee > float64(p.MaxDailyRate) {
			totalFee = float64(p.MaxDailyRate)
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
