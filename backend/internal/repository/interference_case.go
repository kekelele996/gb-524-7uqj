package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type CaseRepository struct {
	db *gorm.DB
}

func NewCaseRepository(db *gorm.DB) *CaseRepository {
	return &CaseRepository{db: db}
}

func (r *CaseRepository) List(ctx context.Context, page, pageSize int, status string) ([]model.InterferenceCase, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	query := r.db.WithContext(ctx).Model(&model.InterferenceCase{})
	if status != "" {
		query = query.Where("case_status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count interference cases: %w", err)
	}
	var cases []model.InterferenceCase
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&cases).Error; err != nil {
		return nil, 0, fmt.Errorf("list interference cases: %w", err)
	}
	return cases, total, nil
}

func (r *CaseRepository) Get(ctx context.Context, id uint) (model.InterferenceCase, error) {
	var item model.InterferenceCase
	if err := r.db.WithContext(ctx).First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.InterferenceCase{}, api.NewError(404, "CASE_NOT_FOUND", "干扰案例不存在")
		}
		return model.InterferenceCase{}, fmt.Errorf("get interference case: %w", err)
	}
	return item, nil
}

func (r *CaseRepository) Create(ctx context.Context, item *model.InterferenceCase, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				return api.NewError(409, "CASE_CODE_EXISTS", "案例编号已存在")
			}
			return fmt.Errorf("create interference case: %w", err)
		}
		audit := NewAudit(actor, "interference_case.created", "interference_case", item.ID, nil, item)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit case create: %w", err)
		}
		return nil
	})
}

func (r *CaseRepository) Transition(ctx context.Context, id uint, version uint, target constants.CaseStatus, conclusion, reason string, reviewerID *uint, actor Actor) (model.InterferenceCase, error) {
	var updated model.InterferenceCase
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.InterferenceCase
		if err := tx.First(&before, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return api.NewError(404, "CASE_NOT_FOUND", "干扰案例不存在")
			}
			return fmt.Errorf("load case for transition: %w", err)
		}
		if before.Version != version {
			return api.NewError(409, "CASE_VERSION_CONFLICT", "案例版本已更新，请刷新后重试")
		}
		if !constants.CanTransitionCase(before.CaseStatus, target) {
			return api.WithDetails(api.NewError(409, "INVALID_CASE_TRANSITION", "当前案例状态不允许该迁移"), map[string]any{
				"current": before.CaseStatus, "target": target,
			})
		}
		updates := map[string]any{
			"case_status": target, "version": gorm.Expr("version + 1"),
			"conclusion": conclusion, "review_reason": reason,
		}
		if reviewerID != nil {
			updates["reviewer_id"] = *reviewerID
		}
		if target == constants.CaseClosed {
			now := time.Now().UTC()
			updates["closed_at"] = &now
		}
		result := tx.Model(&model.InterferenceCase{}).Where("id = ? AND version = ? AND case_status = ?", id, version, before.CaseStatus).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition interference case: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return api.NewError(409, "CASE_VERSION_CONFLICT", "案例被其他请求更新，请刷新后重试")
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload transitioned case: %w", err)
		}
		action := "interference_case." + string(target)
		audit := NewAudit(actor, action, "interference_case", id, before, updated)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit case transition: %w", err)
		}
		return nil
	})
	return updated, err
}

func (r *CaseRepository) Counts(ctx context.Context, caseID uint) (observations, active, estimates int64, err error) {
	if err = r.db.WithContext(ctx).Model(&model.BearingObservation{}).Where("case_id = ?", caseID).Count(&observations).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("count case observations: %w", err)
	}
	if err = r.db.WithContext(ctx).Model(&model.BearingObservation{}).Where("case_id = ? AND quality <> ?", caseID, constants.QualityExcluded).Count(&active).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("count active observations: %w", err)
	}
	if err = r.db.WithContext(ctx).Model(&model.LocalizationEstimate{}).Where("case_id = ?", caseID).Count(&estimates).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("count estimates: %w", err)
	}
	return observations, active, estimates, nil
}
