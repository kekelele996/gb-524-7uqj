package dto

import "time"

type CreateStationRequest struct {
	StationCode    string     `json:"station_code" binding:"required,min=2,max=32"`
	Name           string     `json:"name" binding:"required,min=2,max=120"`
	Latitude       float64    `json:"latitude" binding:"gte=-90,lte=90"`
	Longitude      float64    `json:"longitude" binding:"gte=-180,lte=180"`
	AntennaBiasDeg float64    `json:"antenna_bias_deg" binding:"gte=-30,lte=30"`
	AccuracyDeg    float64    `json:"accuracy_deg" binding:"required,gt=0,lte=45"`
	StationStatus  string     `json:"station_status" binding:"required,oneof=active calibration_due inactive"`
	CalibratedAt   *time.Time `json:"calibrated_at"`
}

type UpdateStationRequest struct {
	Name           string     `json:"name" binding:"required,min=2,max=120"`
	Latitude       float64    `json:"latitude" binding:"gte=-90,lte=90"`
	Longitude      float64    `json:"longitude" binding:"gte=-180,lte=180"`
	AntennaBiasDeg float64    `json:"antenna_bias_deg" binding:"gte=-30,lte=30"`
	AccuracyDeg    float64    `json:"accuracy_deg" binding:"required,gt=0,lte=45"`
	StationStatus  string     `json:"station_status" binding:"required,oneof=active calibration_due inactive"`
	CalibratedAt   *time.Time `json:"calibrated_at"`
}

type StationCoverage struct {
	StationID        uint       `json:"station_id"`
	ObservationCount int64      `json:"observation_count"`
	LastObservedAt   *time.Time `json:"last_observed_at"`
}
