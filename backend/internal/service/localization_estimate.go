package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/localization"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
	"spectrum-interference-triangulation/backend/internal/util"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type EstimateService struct {
	repo            *repository.EstimateRepository
	observationRepo *repository.ObservationRepository
	caseRepo        *repository.CaseRepository
	conditionLimit  float64
}

type RunResult struct {
	Primary   model.LocalizationEstimate  `json:"primary"`
	Candidate *model.LocalizationEstimate `json:"candidate,omitempty"`
}

// ReintersectionSummary 汇总站点变更后对其未结案案例的自动重新交汇情况，
// 通过更新站点接口的 meta 返回给前端。
type ReintersectionSummary struct {
	StationID                    uint                 `json:"station_id"`
	LocalizationChanged          bool                 `json:"localization_changed"`
	AffectedCaseIDs              []uint               `json:"affected_case_ids"`
	RecalibratedObservationCount int                  `json:"recalibrated_observation_count"`
	ReintersectedCaseIDs         []uint               `json:"reintersected_case_ids"`
	SkippedCases                 []ReintersectionSkip `json:"skipped_cases"`
}

// ReintersectionSkip 记录无法自动重新交汇的案例及原因代码。
type ReintersectionSkip struct {
	CaseID uint   `json:"case_id"`
	Reason string `json:"reason"`
}

const (
	skipCaseFinalized      = "case_finalized"
	skipNoPriorEstimate    = "no_prior_estimate"
	skipFrequencyMismatch  = "frequency_mismatch"
	skipInsufficientInputs = "insufficient_observations"
	skipGeometryDegenerate = "geometry_degenerate"
)

func NewEstimateService(repo *repository.EstimateRepository, observationRepo *repository.ObservationRepository, caseRepo *repository.CaseRepository, conditionLimit float64) *EstimateService {
	return &EstimateService{repo: repo, observationRepo: observationRepo, caseRepo: caseRepo, conditionLimit: conditionLimit}
}

func (s *EstimateService) List(ctx context.Context, caseID uint) ([]model.LocalizationEstimate, error) {
	return s.repo.List(ctx, caseID)
}

func (s *EstimateService) Get(ctx context.Context, id uint) (model.LocalizationEstimate, error) {
	return s.repo.Get(ctx, id)
}

func (s *EstimateService) Run(ctx context.Context, request dto.RunLocalizationRequest, actor repository.Actor) (RunResult, error) {
	if !constants.CanAnalyze(actor.Role) {
		return RunResult{}, api.ErrForbidden
	}
	caseRecord, err := s.caseRepo.Get(ctx, request.CaseID)
	if err != nil {
		return RunResult{}, err
	}
	if caseRecord.CaseStatus != constants.CaseAnalyzing {
		return RunResult{}, api.WithDetails(api.NewError(409, "CASE_NOT_ANALYZING", "只有 analyzing 状态的案例可以运行定位"), map[string]any{"current": caseRecord.CaseStatus})
	}
	observations, err := s.observationRepo.ListForCase(ctx, request.CaseID, false)
	if err != nil {
		return RunResult{}, err
	}
	inputs, err := buildSolverInputs(caseRecord.FrequencyCenterHz, observations, true)
	if err != nil {
		return RunResult{}, err
	}
	run, err := solveRun(inputs, s.conditionLimit, request.AllowOutlier)
	if err != nil {
		return RunResult{}, err
	}
	primary, err := buildEstimate(caseRecord.ID, actor.UserID, constants.EstimateComplete, run.Primary, inputs)
	if err != nil {
		return RunResult{}, err
	}
	var candidate *model.LocalizationEstimate
	if run.Candidate != nil {
		candidateValue, buildErr := buildEstimate(caseRecord.ID, actor.UserID, constants.EstimateCandidate, *run.Candidate, inputs)
		if buildErr != nil {
			return RunResult{}, buildErr
		}
		candidate = &candidateValue
	}
	if err := s.repo.CreateRun(ctx, caseRecord.ID, caseRecord.Version, &primary, candidate, request.AllowOutlier, s.conditionLimit, actor); err != nil {
		return RunResult{}, err
	}
	return RunResult{Primary: primary, Candidate: candidate}, nil
}

// buildSolverInputs 按案例中心频率校验各观测频率，并使用站点当前坐标、
// 精度和观测的校正方位构造求解器输入；未启用站点的观测静默不参与交汇。
// enforceFrequency=false 时（站点变更后的自动重算），频率不匹配以第二个返回值报告，
// 由调用方决定是否跳过该案例，而不是中断整个站点更新事务。
func buildSolverInputs(centerHz float64, observations []model.BearingObservation, enforceFrequency bool) ([]localization.Input, error) {
	inputs := make([]localization.Input, 0, len(observations))
	for _, observation := range observations {
		if observation.Station == nil || observation.Station.StationStatus != "active" {
			continue
		}
		if err := validateFrequency(centerHz, observation.FrequencyHz, observation.BandwidthHz); err != nil {
			if enforceFrequency {
				return nil, err
			}
			return nil, errFrequencyMismatch
		}
		inputs = append(inputs, localization.Input{
			ObservationID: observation.ID, StationCode: observation.Station.StationCode,
			Latitude: observation.Station.Latitude, Longitude: observation.Station.Longitude,
			BearingDeg: observation.CorrectedBearingDeg, AccuracyDeg: observation.Station.AccuracyDeg,
			QualityWeight: constants.QualityWeight(observation.Quality),
		})
	}
	return inputs, nil
}

var errFrequencyMismatch = errors.New("observation frequency outside case bandwidth")

// solveRun 执行确定性求解并把算法错误翻译成业务错误（手动运行使用）。
func solveRun(inputs []localization.Input, conditionLimit float64, allowOutlier bool) (localization.Run, error) {
	run, err := localization.SolveWithOutlierCandidate(inputs, conditionLimit, allowOutlier)
	if err != nil {
		var degenerate *localization.DegenerateError
		if errors.As(err, &degenerate) {
			return localization.Run{}, api.WithDetails(api.NewError(422, "GEOMETRY_DEGENERATE", "方位几何退化，无法形成可信定位点"), map[string]any{
				"condition_number": util.JSONSafeNumber(degenerate.ConditionNumber), "reason": degenerate.Reason,
			})
		}
		if errors.Is(err, localization.ErrInsufficientObservations) {
			return localization.Run{}, api.NewError(422, "INSUFFICIENT_OBSERVATIONS", "定位至少需要两条来自启用测向站的有效观测")
		}
		return localization.Run{}, fmt.Errorf("solve localization: %w", err)
	}
	return run, nil
}

// ReintersectOpenCases 在站点校准更新事务内执行传播：
// 以站点当前偏置重算该站在所有未结案案例中的观测校正方位，
// 并对其中已有定位结果的案例用站点当前坐标/精度/偏置追加一次重新交汇。
// confirmed/closed 案例不查询、不重算、不追加，历史证据保持冻结。
func (s *EstimateService) ReintersectOpenCases(ctx context.Context, tx *gorm.DB, stationID uint, actor repository.Actor) (ReintersectionSummary, error) {
	summary := ReintersectionSummary{
		StationID:            stationID,
		LocalizationChanged:  true,
		AffectedCaseIDs:      []uint{},
		ReintersectedCaseIDs: []uint{},
		SkippedCases:         []ReintersectionSkip{},
	}
	stationObservations, err := s.observationRepo.ListOpenCaseObservationsForStation(ctx, tx, stationID)
	if err != nil {
		return ReintersectionSummary{}, err
	}
	caseIDs, err := s.caseRepo.OpenCaseIDsForStation(ctx, tx, stationID)
	if err != nil {
		return ReintersectionSummary{}, err
	}
	summary.AffectedCaseIDs = caseIDs

	correctedByID := make(map[uint]float64, len(stationObservations))
	for i := range stationObservations {
		observation := stationObservations[i]
		if observation.Station == nil {
			continue
		}
		correctedByID[observation.ID] = normalizeBearing(observation.BearingDeg + observation.Station.AntennaBiasDeg)
	}
	changed, err := s.observationRepo.RecalibrateObservations(ctx, tx, stationObservations, correctedByID, actor)
	if err != nil {
		return ReintersectionSummary{}, err
	}
	summary.RecalibratedObservationCount = changed

	for _, caseID := range caseIDs {
		caseRecord, err := s.caseRepo.GetTx(ctx, tx, caseID)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		if !constants.IsOpenCaseStatus(caseRecord.CaseStatus) {
			// 并发保护：读取后案例已被确认或关闭，维持冻结。
			summary.SkippedCases = append(summary.SkippedCases, ReintersectionSkip{CaseID: caseID, Reason: skipCaseFinalized})
			continue
		}
		estimateCount, err := s.caseRepo.CountEstimatesTx(ctx, tx, caseID)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		if estimateCount == 0 {
			// 从未交汇过的案例不自动补跑；分析员将案例推进到 analyzing 后手动运行即可。
			summary.SkippedCases = append(summary.SkippedCases, ReintersectionSkip{CaseID: caseID, Reason: skipNoPriorEstimate})
			continue
		}
		observations, err := s.observationRepo.ListForCaseTx(ctx, tx, caseID, false)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		inputs, inputErr := buildSolverInputs(caseRecord.FrequencyCenterHz, observations, false)
		if inputErr != nil {
			skipErr := s.skipReintersection(ctx, tx, caseID, stationID, actor, inputErr, nil)
			if skipErr != nil {
				return ReintersectionSummary{}, skipErr
			}
			summary.SkippedCases = append(summary.SkippedCases, ReintersectionSkip{CaseID: caseID, Reason: reasonForSolverError(inputErr)})
			continue
		}
		run, solveErr := localization.SolveWithOutlierCandidate(inputs, s.conditionLimit, true)
		if solveErr != nil {
			detail := map[string]any{}
			var degenerate *localization.DegenerateError
			if errors.As(solveErr, &degenerate) {
				detail["condition_number"] = util.JSONSafeNumber(degenerate.ConditionNumber)
				detail["solver_reason"] = degenerate.Reason
			}
			if err := s.skipReintersection(ctx, tx, caseID, stationID, actor, solveErr, detail); err != nil {
				return ReintersectionSummary{}, err
			}
			summary.SkippedCases = append(summary.SkippedCases, ReintersectionSkip{CaseID: caseID, Reason: reasonForSolverError(solveErr)})
			continue
		}
		primary, err := buildEstimate(caseID, actor.UserID, constants.EstimateComplete, run.Primary, inputs)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		var candidate *model.LocalizationEstimate
		if run.Candidate != nil {
			candidateValue, buildErr := buildEstimate(caseID, actor.UserID, constants.EstimateCandidate, *run.Candidate, inputs)
			if buildErr != nil {
				return ReintersectionSummary{}, buildErr
			}
			candidate = &candidateValue
		}
		previousID, err := s.repo.LatestIDForCaseTx(ctx, tx, caseID)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		saved, err := s.repo.SaveReintersection(ctx, tx, repository.ReintersectionAudit{
			CaseID: caseID, Version: caseRecord.Version, PreviousID: previousID,
			ConditionLimit: s.conditionLimit,
		}, &primary, candidate, actor)
		if err != nil {
			return ReintersectionSummary{}, err
		}
		if !saved {
			summary.SkippedCases = append(summary.SkippedCases, ReintersectionSkip{CaseID: caseID, Reason: skipCaseFinalized})
			continue
		}
		summary.ReintersectedCaseIDs = append(summary.ReintersectedCaseIDs, caseID)
	}
	return summary, nil
}

func (s *EstimateService) skipReintersection(ctx context.Context, tx *gorm.DB, caseID, stationID uint, actor repository.Actor, cause error, detail map[string]any) error {
	reason := reasonForSolverError(cause)
	return s.repo.LogReintersectionSkipped(ctx, tx, caseID, stationID, reason, detail, actor)
}

func reasonForSolverError(err error) string {
	switch {
	case errors.Is(err, errFrequencyMismatch):
		return skipFrequencyMismatch
	case errors.Is(err, localization.ErrInsufficientObservations):
		return skipInsufficientInputs
	default:
		var degenerate *localization.DegenerateError
		if errors.As(err, &degenerate) {
			return skipGeometryDegenerate
		}
		return skipGeometryDegenerate
	}
}

func buildEstimate(caseID, userID uint, status string, result localization.Result, inputs []localization.Input) (model.LocalizationEstimate, error) {
	usedJSON, err := json.Marshal(result.UsedObservationIDs)
	if err != nil {
		return model.LocalizationEstimate{}, fmt.Errorf("marshal used observation IDs: %w", err)
	}
	outlierJSON, err := json.Marshal(result.OutlierIDs)
	if err != nil {
		return model.LocalizationEstimate{}, fmt.Errorf("marshal outlier IDs: %w", err)
	}
	residualJSON, err := json.Marshal(result.Residuals)
	if err != nil {
		return model.LocalizationEstimate{}, fmt.Errorf("marshal residual evidence: %w", err)
	}
	snapshotJSON, err := json.Marshal(inputs)
	if err != nil {
		return model.LocalizationEstimate{}, fmt.Errorf("marshal localization snapshot: %w", err)
	}
	if math.IsNaN(result.Point.Latitude) || math.IsNaN(result.Point.Longitude) {
		return model.LocalizationEstimate{}, api.NewError(422, "INVALID_ESTIMATE", "定位算法产生了无效坐标")
	}
	return model.LocalizationEstimate{
		CaseID: caseID, AlgorithmVersion: localization.AlgorithmVersion,
		Latitude: result.Point.Latitude, Longitude: result.Point.Longitude,
		UncertaintyRadiusM: result.UncertaintyRadiusM, ResidualDeg: result.ResidualDeg,
		ConditionNumber: result.ConditionNumber, GeometryDegenerate: false,
		UsedObservationIDsJSON: datatypes.JSON(usedJSON), OutlierIDsJSON: datatypes.JSON(outlierJSON),
		ResidualsJSON: datatypes.JSON(residualJSON), InputSnapshotJSON: datatypes.JSON(snapshotJSON),
		EstimateStatus: status, CreatedBy: userID,
	}, nil
}
