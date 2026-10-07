package model

import (
	"time"

	"spectrum-interference-triangulation/backend/internal/constants"
)

type InterferenceCase struct {
	ID                uint                 `json:"id" gorm:"primaryKey"`
	CaseCode          string               `json:"case_code" gorm:"size:40;not null;uniqueIndex"`
	Title             string               `json:"title" gorm:"size:160;not null"`
	FrequencyCenterHz float64              `json:"frequency_center_hz" gorm:"not null;check:frequency_center_hz > 0"`
	CaseStatus        constants.CaseStatus `json:"case_status" gorm:"type:varchar(24);not null;default:draft;check:case_status IN ('draft','collecting','analyzing','pending_review','confirmed','closed')"`
	Priority          string               `json:"priority" gorm:"size:16;not null;default:normal;check:priority IN ('low','normal','high')"`
	OpenedBy          uint                 `json:"opened_by" gorm:"not null"`
	ReviewerID        *uint                `json:"reviewer_id"`
	Conclusion        string               `json:"conclusion" gorm:"size:2000"`
	ReviewReason      string               `json:"review_reason" gorm:"size:1000"`
	Version           uint                 `json:"version" gorm:"not null;default:1"`
	ClosedAt          *time.Time           `json:"closed_at"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

func (InterferenceCase) TableName() string { return "interference_cases" }
