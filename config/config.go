package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	DB      DBConfig      `yaml:"database"`
	JWT     JWTConfig     `yaml:"jwt"`
	Parking ParkingConfig `yaml:"parking"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf(":%d", s.Port)
}

type DBConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	DBName    string `yaml:"dbname"`
	Charset   string `yaml:"charset"`
	ParseTime bool   `yaml:"parse_time"`
	Loc       string `yaml:"loc"`
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%v&loc=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset, d.ParseTime, d.Loc)
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type ParkingConfig struct {
	TotalSpaces      int `yaml:"total_spaces"`
	HourlyRate       int `yaml:"hourly_rate"`
	DaytimeRate      int `yaml:"daytime_rate"`
	NighttimeRate    int `yaml:"nighttime_rate"`
	DaytimeStart     int `yaml:"daytime_start"`
	DaytimeEnd       int `yaml:"daytime_end"`
	MaxDailyRate     int `yaml:"max_daily_rate"`
	FreeMinutes      int `yaml:"free_minutes"`
	MonthlyCardPrice int `yaml:"monthly_card_price"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
