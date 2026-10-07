package dto

type RunLocalizationRequest struct {
	CaseID       uint `json:"case_id" binding:"required"`
	AllowOutlier bool `json:"allow_outlier"`
}

type ResidualEvidence struct {
	ObservationID uint    `json:"observation_id"`
	StationCode   string  `json:"station_code"`
	ObservedDeg   float64 `json:"observed_deg"`
	PredictedDeg  float64 `json:"predicted_deg"`
	ResidualDeg   float64 `json:"residual_deg"`
	Standardized  float64 `json:"standardized"`
}

type LocalizationRunResponse struct {
	Primary        any  `json:"primary"`
	Candidate      any  `json:"candidate,omitempty"`
	OutlierRemoved bool `json:"outlier_removed"`
}
