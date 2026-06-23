package services

import (
	"parking-system/config"
	"testing"
	"time"
)

func testCfg() *config.Config {
	return &config.Config{
		Parking: config.ParkingConfig{
			DaytimeRate:      6,
			NighttimeRate:    3,
			DaytimeStart:     8,
			DaytimeEnd:       20,
			MaxDailyRate:     50,
			FreeMinutes:      15,
			MonthlyCardPrice: 200,
		},
	}
}

func TestSameDayParking(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 15, 14, 30, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)

	if bd.DayDuration != 255 {
		t.Errorf("DayDuration: got %d, want 255", bd.DayDuration)
	}
	if bd.NightDuration != 0 {
		t.Errorf("NightDuration: got %d, want 0", bd.NightDuration)
	}
	if bd.TotalFee != 30.0 {
		t.Errorf("TotalFee: got %v, want 30", bd.TotalFee)
	}
}

func TestCrossDayParking_6pmTo9am(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 18, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 16, 9, 0, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)

	if bd.DayDuration != 165 {
		t.Errorf("DayDuration: got %d, want 165 (105min day1 + 60min day2)", bd.DayDuration)
	}
	if bd.NightDuration != 720 {
		t.Errorf("NightDuration: got %d, want 720 (240min night1 + 480min night2)", bd.NightDuration)
	}

	day1Fee := 2*6 + 4*3
	day2Fee := 1*6 + 8*3
	expectedTotal := float64(day1Fee + day2Fee)

	if bd.TotalFee != expectedTotal {
		t.Errorf("TotalFee: got %v, want %v", bd.TotalFee, expectedTotal)
	}
}

func TestCrossDayParking_3days(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 18, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 17, 9, 0, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)

	day1Fee := float64(2*6 + 4*3)
	day2Fee := float64(12*6 + 12*3)
	if day2Fee > 50 {
		day2Fee = 50
	}
	day3Fee := float64(1*6 + 8*3)
	expectedTotal := day1Fee + day2Fee + day3Fee

	if bd.TotalFee != expectedTotal {
		t.Errorf("TotalFee: got %v, want %v", bd.TotalFee, expectedTotal)
	}
}

func TestFreeMinutes(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 15, 10, 10, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)
	if bd.TotalFee != 0 {
		t.Errorf("TotalFee: got %v, want 0 (within free minutes)", bd.TotalFee)
	}
}

func TestMonthlyCardFree(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 10, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 16, 10, 0, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, true)
	if bd.TotalFee != 0 {
		t.Errorf("TotalFee: got %v, want 0 (monthly card)", bd.TotalFee)
	}
}

func TestNightOnlyParking(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 21, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 16, 7, 0, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)

	if bd.DayDuration != 0 {
		t.Errorf("DayDuration: got %d, want 0", bd.DayDuration)
	}
	if bd.NightDuration != 585 {
		t.Errorf("NightDuration: got %d, want 585 (660min total - 15min free - 60min daytime 08:00-07:00 overlap)", bd.NightDuration)
	}

	nightHours := 0
	if bd.NightDuration > 0 {
		nightHours = int(float64(bd.NightDuration)/60.0 + 0.999)
	}
	expectedFee := float64(nightHours * 3)
	if expectedFee > 50 {
		expectedFee = 50
	}

	if bd.TotalFee != expectedFee {
		t.Errorf("TotalFee: got %v, want %v", bd.TotalFee, expectedFee)
	}
}

func TestDailyCapPerDay(t *testing.T) {
	svc := NewParkingService(testCfg())
	loc := time.FixedZone("CST", 8*3600)

	entry := time.Date(2024, 1, 15, 8, 0, 0, 0, loc)
	exit := time.Date(2024, 1, 15, 23, 0, 0, 0, loc)

	bd := svc.CalculateFee(entry, exit, false)

	if bd.TotalFee != 50 {
		t.Errorf("TotalFee: got %v, want 50 (daily cap)", bd.TotalFee)
	}
}
