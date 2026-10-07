package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type EstimateRepository struct {
	db *gorm.DB
}

func NewEstimateRepository(db *gorm.DB) *EstimateRepository {
	return &EstimateRepository{db: db}
}

type ReintersectionAudit struct {
	CaseID         uint
	Version        uint
	PreviousID     uint
	ConditionLimit float64
}

// SaveReintersection 在调用方事务中保存站点变更触发的重新交汇结果。
// 旧估计保持不可覆盖：新结果以追加方式写入，案例 version 只在仍未结案时递增；
// 若案例在计算期间被确认或关闭，则放弃写入并返回 false，由上层记跳过审计。
func (r *EstimateRepository) SaveReintersection(ctx context.Context, tx *gorm.DB, auditInfo ReintersectionAudit, primary *model.LocalizationEstimate, candidate *model.LocalizationEstimate, actor Actor) (bool, error) {
	claim := tx.Model(&model.InterferenceCase{}).
		Where("id = ? AND version = ? AND case_status IN ?", auditInfo.CaseID, auditInfo.Version, constants.OpenCaseStatusValues()).
		UpdateColumn("version", gorm.Expr("version + 1"))
	if claim.Error != nil {
		return false, fmt.Errorf("claim case reintersection: %w", claim.Error)
	}
	if claim.RowsAffected != 1 {
		return false, nil
	}
	if err := tx.Create(primary).Error; err != nil {
		return false, fmt.Errorf("save reintersection primary estimate: %w", err)
	}
	if candidate != nil {
		candidate.ParentEstimateID = &primary.ID
		if err := tx.Create(candidate).Error; err != nil {
			return false, fmt.Errorf("save reintersection candidate estimate: %w", err)
		}
	}
	after := map[string]any{
		"primary_id": primary.ID, "previous_estimate_id": auditInfo.PreviousID,
		"trigger": "station_updated", "algorithm_version": primary.AlgorithmVersion,
		"condition_limit":      auditInfo.ConditionLimit,
		"residual_deg":         primary.ResidualDeg,
		"condition_number":     primary.ConditionNumber,
		"used_observation_ids": primary.UsedObservationIDsJSON,
		"outlier_ids":          primary.OutlierIDsJSON,
	}
	audit := NewAudit(actor, "localization_estimate.reintersected", "interference_case", auditInfo.CaseID, map[string]any{"version": auditInfo.Version}, after)
	if err := tx.Create(&audit).Error; err != nil {
		return false, fmt.Errorf("audit reintersection run: %w", err)
	}
	return true, nil
}

// LogReintersectionSkipped 为无法完成自动重新交汇的未结案案例保留一条审计，
// 旧估计与案例状态都不改动，分析员可据此在定位页手动重跑。
func (r *EstimateRepository) LogReintersectionSkipped(ctx context.Context, tx *gorm.DB, caseID uint, stationID uint, reason string, detail map[string]any, actor Actor) error {
	after := map[string]any{"trigger": "station_updated", "station_id": stationID, "reason": reason}
	for key, value := range detail {
		after[key] = value
	}
	audit := NewAudit(actor, "localization_estimate.reintersection_skipped", "interference_case", caseID, map[string]any{}, after)
	if err := tx.Create(&audit).Error; err != nil {
		return fmt.Errorf("audit reintersection skip: %w", err)
	}
	return nil
}

func (r *EstimateRepository) List(ctx context.Context, caseID uint) ([]model.LocalizationEstimate, error) {
	query := r.db.WithContext(ctx).Model(&model.LocalizationEstimate{})
	if caseID > 0 {
		query = query.Where("case_id = ?", caseID)
	}
	var estimates []model.LocalizationEstimate
	if err := query.Order("created_at DESC, id DESC").Find(&estimates).Error; err != nil {
		return nil, fmt.Errorf("list localization estimates: %w", err)
	}
	return estimates, nil
}

func (r *EstimateRepository) Get(ctx context.Context, id uint) (model.LocalizationEstimate, error) {
	var estimate model.LocalizationEstimate
	if err := r.db.WithContext(ctx).First(&estimate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.LocalizationEstimate{}, api.NewError(404, "ESTIMATE_NOT_FOUND", "定位结果不存在")
		}
		return model.LocalizationEstimate{}, fmt.Errorf("get localization estimate: %w", err)
	}
	return estimate, nil
}

// LatestIDForCaseTx 在调用方事务中返回案例最近一条定位结果的 ID（无记录返回 0），
// 用于重新交汇审计里串联新旧估计。
func (r *EstimateRepository) LatestIDForCaseTx(ctx context.Context, tx *gorm.DB, caseID uint) (uint, error) {
	var id uint
	err := tx.WithContext(ctx).Model(&model.LocalizationEstimate{}).
		Where("case_id = ?", caseID).
		Order("created_at DESC, id DESC").
		Limit(1).
		Pluck("id", &id).Error
	if err != nil {
		return 0, fmt.Errorf("get latest estimate id: %w", err)
	}
	return id, nil
}

func (r *EstimateRepository) CreateRun(ctx context.Context, caseID, version uint, primary *model.LocalizationEstimate, candidate *model.LocalizationEstimate, allowOutlier bool, conditionLimit float64, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&model.InterferenceCase{}).
			Where("id = ? AND version = ? AND case_status = ?", caseID, version, constants.CaseAnalyzing).
			UpdateColumn("version", gorm.Expr("version + 1"))
		if claim.Error != nil {
			return fmt.Errorf("claim case localization run: %w", claim.Error)
		}
		if claim.RowsAffected != 1 {
			return api.NewError(409, "CASE_VERSION_CONFLICT", "案例状态或版本已变化，请刷新后重试")
		}
		if err := tx.Create(primary).Error; err != nil {
			return fmt.Errorf("save primary estimate: %w", err)
		}
		if candidate != nil {
			candidate.ParentEstimateID = &primary.ID
			if err := tx.Create(candidate).Error; err != nil {
				return fmt.Errorf("save outlier candidate estimate: %w", err)
			}
		}
		after := map[string]any{
			"primary_id":        primary.ID,
			"algorithm_version": primary.AlgorithmVersion,
			"allow_outlier":     allowOutlier,
			"condition_limit":   conditionLimit,
			"candidate_id": func() uint {
				if candidate == nil {
					return 0
				}
				return candidate.ID
			}(),
			"residual_deg":         primary.ResidualDeg,
			"condition_number":     primary.ConditionNumber,
			"geometry_degenerate":  primary.GeometryDegenerate,
			"used_observation_ids": primary.UsedObservationIDsJSON,
			"outlier_ids":          primary.OutlierIDsJSON,
		}
		audit := NewAudit(actor, "localization_estimate.created", "interference_case", caseID, map[string]any{"version": version}, after)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit localization run: %w", err)
		}
		return nil
	})
}
