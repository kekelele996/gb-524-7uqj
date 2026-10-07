package dto

import "spectrum-interference-triangulation/backend/internal/constants"

type CreateCaseRequest struct {
	CaseCode          string  `json:"case_code" binding:"required,min=4,max=40"`
	Title             string  `json:"title" binding:"required,min=4,max=160"`
	FrequencyCenterHz float64 `json:"frequency_center_hz" binding:"required,gt=0"`
	Priority          string  `json:"priority" binding:"required,oneof=low normal high"`
}

type TransitionCaseRequest struct {
	TargetStatus constants.CaseStatus `json:"target_status" binding:"required"`
	Version      uint                 `json:"version" binding:"required"`
	Conclusion   string               `json:"conclusion" binding:"max=2000"`
	Reason       string               `json:"reason" binding:"max=1000"`
}

type CaseSummary struct {
	ID                     uint                 `json:"id"`
	CaseCode               string               `json:"case_code"`
	Title                  string               `json:"title"`
	FrequencyCenterHz      float64              `json:"frequency_center_hz"`
	CaseStatus             constants.CaseStatus `json:"case_status"`
	Priority               string               `json:"priority"`
	Version                uint                 `json:"version"`
	ObservationCount       int64                `json:"observation_count"`
	ActiveObservationCount int64                `json:"active_observation_count"`
	EstimateCount          int64                `json:"estimate_count"`
	Conclusion             string               `json:"conclusion"`
}
