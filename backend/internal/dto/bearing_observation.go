package dto

import "time"

type CreateObservationRequest struct {
	StationID   uint       `json:"station_id" binding:"required"`
	CaseID      uint       `json:"case_id" binding:"required"`
	BearingDeg  float64    `json:"bearing_deg" binding:"gte=0,lt=360"`
	SignalDBM   float64    `json:"signal_dbm" binding:"gte=-200,lte=50"`
	FrequencyHz float64    `json:"frequency_hz" binding:"required,gt=0"`
	BandwidthHz float64    `json:"bandwidth_hz" binding:"required,gt=0"`
	ObservedAt  *time.Time `json:"observed_at"`
	Quality     string     `json:"quality" binding:"required,oneof=good fair poor"`
}

type ExcludeObservationRequest struct {
	Reason string `json:"reason" binding:"required,min=6,max=500"`
}

type ObservationValidation struct {
	ObservationID    uint     `json:"observation_id"`
	Valid            bool     `json:"valid"`
	Issues           []string `json:"issues"`
	FrequencyDeltaHz float64  `json:"frequency_delta_hz"`
}

type BatchValidationResponse struct {
	CaseID  uint                    `json:"case_id"`
	Valid   int                     `json:"valid"`
	Invalid int                     `json:"invalid"`
	Items   []ObservationValidation `json:"items"`
}
