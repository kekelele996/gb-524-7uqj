package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type ObservationFilter struct {
	CaseID    uint
	StationID uint
	Quality   string
	Page      int
	PageSize  int
}

type ObservationRepository struct {
	db *gorm.DB
}

func NewObservationRepository(db *gorm.DB) *ObservationRepository {
	return &ObservationRepository{db: db}
}

func (r *ObservationRepository) List(ctx context.Context, filter ObservationFilter) ([]model.BearingObservation, int64, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).Model(&model.BearingObservation{})
	if filter.CaseID > 0 {
		query = query.Where("case_id = ?", filter.CaseID)
	}
	if filter.StationID > 0 {
		query = query.Where("station_id = ?", filter.StationID)
	}
	if filter.Quality != "" {
		query = query.Where("quality = ?", filter.Quality)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count observations: %w", err)
	}
	var observations []model.BearingObservation
	if err := query.Preload("Station").Order("observed_at DESC").Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&observations).Error; err != nil {
		return nil, 0, fmt.Errorf("list observations: %w", err)
	}
	return observations, total, nil
}

func (r *ObservationRepository) Get(ctx context.Context, id uint) (model.BearingObservation, error) {
	var observation model.BearingObservation
	if err := r.db.WithContext(ctx).Preload("Station").First(&observation, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.BearingObservation{}, api.NewError(404, "OBSERVATION_NOT_FOUND", "观测记录不存在")
		}
		return model.BearingObservation{}, fmt.Errorf("get observation: %w", err)
	}
	return observation, nil
}

func (r *ObservationRepository) ListForCase(ctx context.Context, caseID uint, includeExcluded bool) ([]model.BearingObservation, error) {
	return r.ListForCaseTx(ctx, r.db, caseID, includeExcluded)
}

// ListForCaseTx 是 ListForCase 的事务版本，供跨仓储编排使用。
func (r *ObservationRepository) ListForCaseTx(ctx context.Context, tx *gorm.DB, caseID uint, includeExcluded bool) ([]model.BearingObservation, error) {
	query := tx.WithContext(ctx).Where("case_id = ?", caseID)
	if !includeExcluded {
		query = query.Where("quality <> ?", constants.QualityExcluded)
	}
	var observations []model.BearingObservation
	if err := query.Preload("Station").Order("observed_at ASC").Find(&observations).Error; err != nil {
		return nil, fmt.Errorf("list case observations: %w", err)
	}
	return observations, nil
}

// ListOpenCaseObservationsForStation 取指定测向站在所有未结案案例中的观测（含已排除），
// 并预加载所属案例与站点，供站点变更后重算校正方位使用。
func (r *ObservationRepository) ListOpenCaseObservationsForStation(ctx context.Context, tx *gorm.DB, stationID uint) ([]model.BearingObservation, error) {
	var observations []model.BearingObservation
	err := tx.WithContext(ctx).
		Preload("Station").
		Joins("JOIN interference_cases ON interference_cases.id = bearing_observations.case_id").
		Where("bearing_observations.station_id = ?", stationID).
		Where("interference_cases.case_status IN ?", constants.OpenCaseStatusValues()).
		Order("bearing_observations.case_id ASC, bearing_observations.observed_at ASC").
		Find(&observations).Error
	if err != nil {
		return nil, fmt.Errorf("list open case observations for station: %w", err)
	}
	return observations, nil
}

// RecalibrateObservations 在调用方事务中按站点新天线偏置重算校正方位，
// 仅对实际发生变化的观测写库并逐条保留不可变审计，返回受影响的观测数。
func (r *ObservationRepository) RecalibrateObservations(ctx context.Context, tx *gorm.DB, observations []model.BearingObservation, correctedByID map[uint]float64, actor Actor) (int, error) {
	changed := 0
	for i := range observations {
		observation := observations[i]
		newCorrected, ok := correctedByID[observation.ID]
		if !ok || newCorrected == observation.CorrectedBearingDeg {
			continue
		}
		before := map[string]any{
			"id": observation.ID, "case_id": observation.CaseID,
			"bearing_deg": observation.BearingDeg, "corrected_bearing_deg": observation.CorrectedBearingDeg,
		}
		result := tx.Model(&model.BearingObservation{}).
			Where("id = ? AND corrected_bearing_deg = ?", observation.ID, observation.CorrectedBearingDeg).
			Update("corrected_bearing_deg", newCorrected)
		if result.Error != nil {
			return 0, fmt.Errorf("recalibrate observation %d: %w", observation.ID, result.Error)
		}
		if result.RowsAffected == 0 {
			return 0, api.ErrConflict
		}
		after := map[string]any{
			"id": observation.ID, "case_id": observation.CaseID,
			"bearing_deg": observation.BearingDeg, "corrected_bearing_deg": newCorrected,
		}
		audit := NewAudit(actor, "bearing_observation.recalibrated", "bearing_observation", observation.ID, before, after)
		if err := tx.Create(&audit).Error; err != nil {
			return 0, fmt.Errorf("audit observation recalibration: %w", err)
		}
		changed++
	}
	return changed, nil
}

func (r *ObservationRepository) Create(ctx context.Context, observation *model.BearingObservation, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.InterferenceCase{}).Where("id = ? AND case_status <> ?", observation.CaseID, constants.CaseClosed).Count(&count).Error; err != nil {
			return fmt.Errorf("check observation case: %w", err)
		}
		if count == 0 {
			return api.NewError(409, "CASE_READ_ONLY", "案例不存在或已关闭，不能新增观测")
		}
		if err := tx.Create(observation).Error; err != nil {
			return fmt.Errorf("create observation: %w", err)
		}
		audit := NewAudit(actor, "bearing_observation.created", "bearing_observation", observation.ID, nil, observation)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit observation create: %w", err)
		}
		return nil
	})
}

func (r *ObservationRepository) Exclude(ctx context.Context, id uint, reason string, actor Actor) (model.BearingObservation, error) {
	var updated model.BearingObservation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.BearingObservation
		if err := tx.First(&before, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return api.NewError(404, "OBSERVATION_NOT_FOUND", "观测记录不存在")
			}
			return fmt.Errorf("load observation for exclusion: %w", err)
		}
		if before.Quality == constants.QualityExcluded {
			return api.NewError(409, "OBSERVATION_ALREADY_EXCLUDED", "该观测已经被排除")
		}
		var caseRecord model.InterferenceCase
		if err := tx.First(&caseRecord, before.CaseID).Error; err != nil {
			return fmt.Errorf("load observation case: %w", err)
		}
		if caseRecord.CaseStatus == constants.CaseClosed {
			return api.NewError(409, "CASE_READ_ONLY", "案例已关闭，观测只读")
		}
		result := tx.Model(&model.BearingObservation{}).Where("id = ? AND quality <> ?", id, constants.QualityExcluded).Updates(map[string]any{
			"quality": constants.QualityExcluded, "excluded_reason": reason,
		})
		if result.Error != nil {
			return fmt.Errorf("exclude observation: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return api.ErrConflict
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload excluded observation: %w", err)
		}
		audit := NewAudit(actor, "bearing_observation.excluded", "bearing_observation", id, before, updated)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit observation exclusion: %w", err)
		}
		return nil
	})
	return updated, err
}
