package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type StationRepository struct {
	db *gorm.DB
}

func NewStationRepository(db *gorm.DB) *StationRepository {
	return &StationRepository{db: db}
}

// RecomputeBearing records one corrected bearing that must be refreshed after
// a station calibration change.
type RecomputeBearing struct {
	ObservationID uint    `json:"observation_id"`
	BeforeDeg     float64 `json:"before_deg"`
	AfterDeg      float64 `json:"after_deg"`
}

// CaseRecomputeEntry is the service-built plan for a single still-open case.
// ApplyCalibration re-checks the case status in the transaction so cases
// confirmed or closed concurrently are never modified.
type CaseRecomputeEntry struct {
	CaseID   uint
	CaseCode string
	Version  uint
	Bearings []RecomputeBearing
	Estimate *model.LocalizationEstimate
	SkipCode string
}

// OpenRecomputeCase bundles a case with all of its observations so the service
// can re-apply the updated station values without extra queries.
type OpenRecomputeCase struct {
	Case         model.InterferenceCase
	Observations []model.BearingObservation
}

func finalizedCaseStatuses() []string {
	return []string{string(constants.CaseConfirmed), string(constants.CaseClosed)}
}

// ListOpenRecomputeCases returns every non-finalized case that holds an
// observation from the station, together with that case's observations.
func (r *StationRepository) ListOpenRecomputeCases(ctx context.Context, stationID uint) ([]OpenRecomputeCase, error) {
	var caseIDs []uint
	if err := r.db.WithContext(ctx).Model(&model.BearingObservation{}).
		Distinct("bearing_observations.case_id").
		Joins("JOIN interference_cases ON interference_cases.id = bearing_observations.case_id").
		Where("bearing_observations.station_id = ? AND interference_cases.case_status NOT IN ?", stationID, finalizedCaseStatuses()).
		Pluck("bearing_observations.case_id", &caseIDs).Error; err != nil {
		return nil, fmt.Errorf("list open cases for station recompute: %w", err)
	}
	if len(caseIDs) == 0 {
		return nil, nil
	}
	var cases []model.InterferenceCase
	if err := r.db.WithContext(ctx).Where("id IN ?", caseIDs).Order("id ASC").Find(&cases).Error; err != nil {
		return nil, fmt.Errorf("load open recompute cases: %w", err)
	}
	var observations []model.BearingObservation
	if err := r.db.WithContext(ctx).Where("case_id IN ?", caseIDs).Preload("Station").Order("id ASC").Find(&observations).Error; err != nil {
		return nil, fmt.Errorf("load recompute observations: %w", err)
	}
	grouped := make(map[uint][]model.BearingObservation, len(cases))
	for _, observation := range observations {
		grouped[observation.CaseID] = append(grouped[observation.CaseID], observation)
	}
	result := make([]OpenRecomputeCase, 0, len(cases))
	for _, item := range cases {
		result = append(result, OpenRecomputeCase{Case: item, Observations: grouped[item.ID]})
	}
	return result, nil
}

// ApplyCalibration persists the station update and, in the same transaction,
// refreshes corrected bearings and appends fresh localization estimates for
// still-open cases. Estimates are never overwritten; finalized cases are
// re-checked against the database and skipped entirely.
func (r *StationRepository) ApplyCalibration(ctx context.Context, station *model.ReceiverStation, before model.ReceiverStation, entries []CaseRecomputeEntry, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.ReceiverStation{}).Where("id = ?", station.ID).Updates(map[string]any{
			"name": station.Name, "latitude": station.Latitude, "longitude": station.Longitude,
			"antenna_bias_deg": station.AntennaBiasDeg, "accuracy_deg": station.AccuracyDeg,
			"station_status": station.StationStatus, "calibrated_at": station.CalibratedAt,
		})
		if result.Error != nil {
			return fmt.Errorf("update receiver station: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return api.NewError(404, "STATION_NOT_FOUND", "测向站不存在")
		}
		stationAudit := NewAudit(actor, "receiver_station.calibrated", "receiver_station", station.ID, before, station)
		if err := tx.Create(&stationAudit).Error; err != nil {
			return fmt.Errorf("audit receiver station update: %w", err)
		}
		for _, entry := range entries {
			claimed := tx.Model(&model.InterferenceCase{}).
				Where("id = ? AND version = ? AND case_status NOT IN ?", entry.CaseID, entry.Version, finalizedCaseStatuses()).
				UpdateColumn("version", gorm.Expr("version + 1"))
			if claimed.Error != nil {
				return fmt.Errorf("claim case %d for recompute: %w", entry.CaseID, claimed.Error)
			}
			if claimed.RowsAffected != 1 {
				// The case was confirmed, closed or otherwise changed after the
				// plan was built: leave its evidence untouched.
				continue
			}
			bearingChanges := make([]map[string]any, 0, len(entry.Bearings))
			for _, bearing := range entry.Bearings {
				updated := tx.Model(&model.BearingObservation{}).
					Where("id = ? AND station_id = ? AND case_id IN (SELECT id FROM interference_cases WHERE case_status NOT IN ?)", bearing.ObservationID, station.ID, finalizedCaseStatuses()).
					Update("corrected_bearing_deg", bearing.AfterDeg)
				if updated.Error != nil {
					return fmt.Errorf("refresh corrected bearing %d: %w", bearing.ObservationID, updated.Error)
				}
				if updated.RowsAffected == 1 && bearing.BeforeDeg != bearing.AfterDeg {
					bearingChanges = append(bearingChanges, map[string]any{
						"observation_id": bearing.ObservationID, "before_deg": bearing.BeforeDeg, "after_deg": bearing.AfterDeg,
					})
				}
			}
			after := map[string]any{
				"trigger": "station_calibration", "station_id": station.ID,
				"corrected_observations": bearingChanges,
			}
			action := "localization_estimate.recompute_skipped"
			if entry.Estimate != nil {
				if err := tx.Create(entry.Estimate).Error; err != nil {
					return fmt.Errorf("save recomputed estimate for case %d: %w", entry.CaseID, err)
				}
				action = "localization_estimate.recomputed"
				after["estimate_id"] = entry.Estimate.ID
				after["latitude"] = entry.Estimate.Latitude
				after["longitude"] = entry.Estimate.Longitude
				after["residual_deg"] = entry.Estimate.ResidualDeg
				after["condition_number"] = entry.Estimate.ConditionNumber
				after["uncertainty_radius_m"] = entry.Estimate.UncertaintyRadiusM
			} else {
				after["skip_reason"] = entry.SkipCode
			}
			caseAudit := NewAudit(actor, action, "interference_case", entry.CaseID,
				map[string]any{"version": entry.Version}, after)
			if err := tx.Create(&caseAudit).Error; err != nil {
				return fmt.Errorf("audit case recompute %d: %w", entry.CaseID, err)
			}
		}
		return nil
	})
}

func (r *StationRepository) List(ctx context.Context, page, pageSize int, status string) ([]model.ReceiverStation, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	query := r.db.WithContext(ctx).Model(&model.ReceiverStation{})
	if status != "" {
		query = query.Where("station_status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count receiver stations: %w", err)
	}
	var stations []model.ReceiverStation
	if err := query.Order("station_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&stations).Error; err != nil {
		return nil, 0, fmt.Errorf("list receiver stations: %w", err)
	}
	return stations, total, nil
}

func (r *StationRepository) Get(ctx context.Context, id uint) (model.ReceiverStation, error) {
	var station model.ReceiverStation
	if err := r.db.WithContext(ctx).First(&station, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.ReceiverStation{}, api.NewError(404, "STATION_NOT_FOUND", "测向站不存在")
		}
		return model.ReceiverStation{}, fmt.Errorf("get receiver station: %w", err)
	}
	return station, nil
}

func (r *StationRepository) Create(ctx context.Context, station *model.ReceiverStation, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(station).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				return api.NewError(409, "STATION_CODE_EXISTS", "测向站编号已存在")
			}
			return fmt.Errorf("create receiver station: %w", err)
		}
		audit := NewAudit(actor, "receiver_station.created", "receiver_station", station.ID, nil, station)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit receiver station create: %w", err)
		}
		return nil
	})
}

func (r *StationRepository) Coverage(ctx context.Context, stationID uint) (int64, *model.BearingObservation, error) {
	query := r.db.WithContext(ctx).Model(&model.BearingObservation{}).Where("station_id = ?", stationID)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, nil, fmt.Errorf("count station observations: %w", err)
	}
	var latest model.BearingObservation
	if err := query.Order("observed_at DESC").First(&latest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return count, nil, nil
		}
		return 0, nil, fmt.Errorf("latest station observation: %w", err)
	}
	return count, &latest, nil
}
