package service

import (
	"context"
	"math"
	"strings"
	"time"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type ObservationService struct {
	repo        *repository.ObservationRepository
	stationRepo *repository.StationRepository
	caseRepo    *repository.CaseRepository
}

func NewObservationService(repo *repository.ObservationRepository, stationRepo *repository.StationRepository, caseRepo *repository.CaseRepository) *ObservationService {
	return &ObservationService{repo: repo, stationRepo: stationRepo, caseRepo: caseRepo}
}

func (s *ObservationService) List(ctx context.Context, filter repository.ObservationFilter) ([]model.BearingObservation, int64, error) {
	if filter.Quality != "" && !constants.ValidObservationQuality(constants.ObservationQuality(filter.Quality)) {
		return nil, 0, api.NewError(400, "INVALID_OBSERVATION_QUALITY", "观测质量筛选值无效")
	}
	return s.repo.List(ctx, filter)
}

func (s *ObservationService) Get(ctx context.Context, id uint) (model.BearingObservation, error) {
	return s.repo.Get(ctx, id)
}

func (s *ObservationService) Create(ctx context.Context, request dto.CreateObservationRequest, actor repository.Actor) (model.BearingObservation, error) {
	station, err := s.stationRepo.Get(ctx, request.StationID)
	if err != nil {
		return model.BearingObservation{}, err
	}
	if station.StationStatus != "active" {
		return model.BearingObservation{}, api.NewError(409, "STATION_NOT_ACTIVE", "只有已校准且启用的测向站可以录入观测")
	}
	caseRecord, err := s.caseRepo.Get(ctx, request.CaseID)
	if err != nil {
		return model.BearingObservation{}, err
	}
	if caseRecord.CaseStatus == constants.CaseClosed {
		return model.BearingObservation{}, api.NewError(409, "CASE_READ_ONLY", "案例已关闭，不能录入观测")
	}
	if err := validateFrequency(caseRecord.FrequencyCenterHz, request.FrequencyHz, request.BandwidthHz); err != nil {
		return model.BearingObservation{}, err
	}
	observedAt := time.Now().UTC()
	if request.ObservedAt != nil {
		observedAt = request.ObservedAt.UTC()
	}
	if observedAt.After(time.Now().UTC().Add(5 * time.Minute)) {
		return model.BearingObservation{}, api.NewError(422, "INVALID_OBSERVATION_TIME", "观测时间不能晚于当前时间")
	}
	corrected := normalizeBearing(request.BearingDeg + station.AntennaBiasDeg)
	observation := model.BearingObservation{
		StationID: request.StationID, CaseID: request.CaseID,
		BearingDeg: request.BearingDeg, CorrectedBearingDeg: corrected,
		SignalDBM: request.SignalDBM, FrequencyHz: request.FrequencyHz,
		BandwidthHz: request.BandwidthHz, ObservedAt: observedAt,
		Quality: constants.ObservationQuality(request.Quality), CreatedBy: actor.UserID,
	}
	if err := s.repo.Create(ctx, &observation, actor); err != nil {
		return model.BearingObservation{}, err
	}
	observation.Station = &station
	return observation, nil
}

func (s *ObservationService) Exclude(ctx context.Context, id uint, request dto.ExcludeObservationRequest, actor repository.Actor) (model.BearingObservation, error) {
	if actor.Role != constants.RoleAnalyst && actor.Role != constants.RoleAdmin {
		return model.BearingObservation{}, api.ErrForbidden
	}
	return s.repo.Exclude(ctx, id, strings.TrimSpace(request.Reason), actor)
}

func (s *ObservationService) ValidateCase(ctx context.Context, caseID uint) (dto.BatchValidationResponse, error) {
	caseRecord, err := s.caseRepo.Get(ctx, caseID)
	if err != nil {
		return dto.BatchValidationResponse{}, err
	}
	observations, err := s.repo.ListForCase(ctx, caseID, true)
	if err != nil {
		return dto.BatchValidationResponse{}, err
	}
	response := dto.BatchValidationResponse{CaseID: caseID, Items: make([]dto.ObservationValidation, 0, len(observations))}
	for _, observation := range observations {
		item := dto.ObservationValidation{ObservationID: observation.ID, Valid: true, Issues: []string{}}
		item.FrequencyDeltaHz = math.Abs(observation.FrequencyHz - caseRecord.FrequencyCenterHz)
		if observation.Quality == constants.QualityExcluded {
			item.Valid = false
			item.Issues = append(item.Issues, "观测已被人工排除")
		}
		if observation.Station == nil || observation.Station.StationStatus != "active" {
			item.Valid = false
			item.Issues = append(item.Issues, "测向站未处于启用状态")
		}
		if item.FrequencyDeltaHz > observation.BandwidthHz/2 {
			item.Valid = false
			item.Issues = append(item.Issues, "观测频率超出案例中心频率带宽")
		}
		if item.Valid {
			response.Valid++
		} else {
			response.Invalid++
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func validateFrequency(center, observed, bandwidth float64) error {
	delta := math.Abs(center - observed)
	if delta > bandwidth/2 {
		return api.WithDetails(api.NewError(422, "FREQUENCY_MISMATCH", "观测频率超出案例中心频率的有效带宽"), map[string]any{
			"center_hz": center, "observed_hz": observed, "bandwidth_hz": bandwidth, "delta_hz": delta,
		})
	}
	return nil
}

func normalizeBearing(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	return value
}
