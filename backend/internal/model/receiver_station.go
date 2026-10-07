package model

import "time"

type ReceiverStation struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	StationCode    string     `json:"station_code" gorm:"size:32;not null;uniqueIndex"`
	Name           string     `json:"name" gorm:"size:120;not null"`
	Latitude       float64    `json:"latitude" gorm:"not null;check:latitude >= -90 AND latitude <= 90"`
	Longitude      float64    `json:"longitude" gorm:"not null;check:longitude >= -180 AND longitude <= 180"`
	AntennaBiasDeg float64    `json:"antenna_bias_deg" gorm:"not null;default:0;check:antenna_bias_deg >= -30 AND antenna_bias_deg <= 30"`
	AccuracyDeg    float64    `json:"accuracy_deg" gorm:"not null;check:accuracy_deg > 0 AND accuracy_deg <= 45"`
	StationStatus  string     `json:"station_status" gorm:"size:16;not null;default:active;check:station_status IN ('active','calibration_due','inactive')"`
	CalibratedAt   *time.Time `json:"calibrated_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (ReceiverStation) TableName() string { return "receiver_stations" }
