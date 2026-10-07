package model

import (
	"time"

	"spectrum-interference-triangulation/backend/internal/constants"
)

type BearingObservation struct {
	ID                  uint                         `json:"id" gorm:"primaryKey"`
	StationID           uint                         `json:"station_id" gorm:"not null;index"`
	CaseID              uint                         `json:"case_id" gorm:"not null;index"`
	BearingDeg          float64                      `json:"bearing_deg" gorm:"not null;check:bearing_deg >= 0 AND bearing_deg < 360"`
	CorrectedBearingDeg float64                      `json:"corrected_bearing_deg" gorm:"not null;check:corrected_bearing_deg >= 0 AND corrected_bearing_deg < 360"`
	SignalDBM           float64                      `json:"signal_dbm" gorm:"not null;check:signal_dbm >= -200 AND signal_dbm <= 50"`
	FrequencyHz         float64                      `json:"frequency_hz" gorm:"not null;check:frequency_hz > 0"`
	BandwidthHz         float64                      `json:"bandwidth_hz" gorm:"not null;check:bandwidth_hz > 0"`
	ObservedAt          time.Time                    `json:"observed_at" gorm:"not null;index"`
	Quality             constants.ObservationQuality `json:"quality" gorm:"type:varchar(16);not null;check:quality IN ('good','fair','poor','excluded')"`
	ExcludedReason      string                       `json:"excluded_reason" gorm:"size:500"`
	CreatedBy           uint                         `json:"created_by" gorm:"not null"`
	CreatedAt           time.Time                    `json:"created_at"`
	UpdatedAt           time.Time                    `json:"updated_at"`
	Station             *ReceiverStation             `json:"station,omitempty" gorm:"foreignKey:StationID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (BearingObservation) TableName() string { return "bearing_observations" }
