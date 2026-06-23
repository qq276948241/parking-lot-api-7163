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
	if totalMin <= p.FreeMinutes {
		return FeeBreakdown{}
	}

	effectiveStart := entry.Add(time.Duration(p.FreeMinutes) * time.Minute)
	if !effectiveStart.Before(exit) {
		return FeeBreakdown{}
	}

	var totalDayMin, totalNightMin int
	var totalFee float64

	cursor := effectiveStart
	for cursor.Before(exit) {
		nextMidnight := time.Date(cursor.Year(), cursor.Month(), cursor.Day()+1, 0, 0, 0, 0, cursor.Location())
		segEnd := nextMidnight
		if segEnd.After(exit) {
			segEnd = exit
		}

		dayMin, nightMin := s.countDayNightMinutes(cursor, segEnd)

		dayHours := 0
		if dayMin > 0 {
			dayHours = int(math.Ceil(float64(dayMin) / 60))
		}
		nightHours := 0
		if nightMin > 0 {
			nightHours = int(math.Ceil(float64(nightMin) / 60))
		}

		rawFee := float64(dayHours*p.DaytimeRate + nightHours*p.NighttimeRate)
		if rawFee > float64(p.MaxDailyRate) {
			rawFee = float64(p.MaxDailyRate)
		}

		totalDayMin += dayMin
		totalNightMin += nightMin
		totalFee += rawFee

		cursor = nextMidnight
	}

	totalDayHours := 0
	if totalDayMin > 0 {
		totalDayHours = int(math.Ceil(float64(totalDayMin) / 60))
	}
	totalNightHours := 0
	if totalNightMin > 0 {
		totalNightHours = int(math.Ceil(float64(totalNightMin) / 60))
	}

	return FeeBreakdown{
		DayDuration:   totalDayMin,
		NightDuration: totalNightMin,
		DayFee:        float64(totalDayHours * p.DaytimeRate),
		NightFee:      float64(totalNightHours * p.NighttimeRate),
		TotalFee:      totalFee,
	}
}

func (s *ParkingService) countDayNightMinutes(start, end time.Time) (int, int) {
	p := s.cfg.Parking
	loc := start.Location()

	midnight := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	dayStartBoundary := time.Date(start.Year(), start.Month(), start.Day(), p.DaytimeStart, 0, 0, 0, loc)
	dayEndBoundary := time.Date(start.Year(), start.Month(), start.Day(), p.DaytimeEnd, 0, 0, 0, loc)
	nextDayMidnight := time.Date(start.Year(), start.Month(), start.Day()+1, 0, 0, 0, 0, loc)

	nightMin := overlapMinutes(start, end, midnight, dayStartBoundary)
	dayMin := overlapMinutes(start, end, dayStartBoundary, dayEndBoundary)
	nightMin += overlapMinutes(start, end, dayEndBoundary, nextDayMidnight)

	return dayMin, nightMin
}

func overlapMinutes(segStart, segEnd, periodStart, periodEnd time.Time) int {
	start := segStart
	if periodStart.After(start) {
		start = periodStart
	}
	end := segEnd
	if periodEnd.Before(end) {
		end = periodEnd
	}
	if !start.Before(end) {
		return 0
	}
	return int(end.Sub(start).Minutes())
}
