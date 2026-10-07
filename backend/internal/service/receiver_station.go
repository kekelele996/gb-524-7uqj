package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type StationService struct {
	repo         *repository.StationRepository
	txManager    *repository.TransactionManager
	estimateSync *EstimateService
}

func NewStationService(
	repo *repository.StationRepository,
	txManager *repository.TransactionManager,
	estimateSync *EstimateService,
) *StationService {
	return &StationService{repo: repo, txManager: txManager, estimateSync: estimateSync}
}

func (s *StationService) List(ctx context.Context, page, pageSize int, status string) ([]model.ReceiverStation, int64, error) {
	if status != "" && status != "active" && status != "calibration_due" && status != "inactive" {
		return nil, 0, api.NewError(400, "INVALID_STATION_STATUS", "测向站状态筛选值无效")
	}
	return s.repo.List(ctx, page, pageSize, status)
}

func (s *StationService) Get(ctx context.Context, id uint) (model.ReceiverStation, error) {
	return s.repo.Get(ctx, id)
}

func (s *StationService) Create(ctx context.Context, request dto.CreateStationRequest, actor repository.Actor) (model.ReceiverStation, error) {
	if err := validateCoordinates(request.Latitude, request.Longitude); err != nil {
		return model.ReceiverStation{}, err
	}
	station := model.ReceiverStation{
		StationCode:    strings.ToUpper(strings.TrimSpace(request.StationCode)),
		Name:           strings.TrimSpace(request.Name),
		Latitude:       request.Latitude,
		Longitude:      request.Longitude,
		AntennaBiasDeg: request.AntennaBiasDeg,
		AccuracyDeg:    request.AccuracyDeg,
		StationStatus:  request.StationStatus,
		CalibratedAt:   request.CalibratedAt,
	}
	if station.StationStatus == "active" && station.CalibratedAt == nil {
		return model.ReceiverStation{}, api.NewError(422, "CALIBRATION_REQUIRED", "启用测向前必须填写最近校准时间")
	}
	if err := s.repo.Create(ctx, &station, actor); err != nil {
		return model.ReceiverStation{}, err
	}
	return station, nil
}

// Update 保存站点校准信息；当影响交汇的站点信息（坐标、天线偏置、精度或启停用状态）
// 变化时，在同一事务内以站点当前值重算该站未结案案例的观测校正方位并追加重新交汇结果，
// 已确认或关闭的案例完全不动。
func (s *StationService) Update(ctx context.Context, id uint, request dto.UpdateStationRequest, actor repository.Actor) (model.ReceiverStation, ReintersectionSummary, error) {
	before, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.ReceiverStation{}, ReintersectionSummary{}, err
	}
	if err := validateCoordinates(request.Latitude, request.Longitude); err != nil {
		return model.ReceiverStation{}, ReintersectionSummary{}, err
	}
	if request.CalibratedAt != nil && request.CalibratedAt.After(time.Now().UTC().Add(5*time.Minute)) {
		return model.ReceiverStation{}, ReintersectionSummary{}, api.NewError(422, "INVALID_CALIBRATION_TIME", "校准时间不能晚于当前时间")
	}
	updated := before
	updated.Name = strings.TrimSpace(request.Name)
	updated.Latitude = request.Latitude
	updated.Longitude = request.Longitude
	updated.AntennaBiasDeg = request.AntennaBiasDeg
	updated.AccuracyDeg = request.AccuracyDeg
	updated.StationStatus = request.StationStatus
	updated.CalibratedAt = request.CalibratedAt
	if updated.StationStatus == "active" && updated.CalibratedAt == nil {
		return model.ReceiverStation{}, ReintersectionSummary{}, api.NewError(422, "CALIBRATION_REQUIRED", "启用测向前必须填写最近校准时间")
	}
	geometryChanged := stationLocalizationChanged(before, updated)

	summary := ReintersectionSummary{
		StationID: id, LocalizationChanged: geometryChanged,
		AffectedCaseIDs: []uint{}, ReintersectedCaseIDs: []uint{}, SkippedCases: []ReintersectionSkip{},
	}
	err = s.txManager.Within(ctx, func(tx *gorm.DB) error {
		if saveErr := s.repo.SaveUpdate(ctx, tx, &updated, before, actor); saveErr != nil {
			return saveErr
		}
		if !geometryChanged {
			return nil
		}
		propagated, propagateErr := s.estimateSync.ReintersectOpenCases(ctx, tx, id, actor)
		if propagateErr != nil {
			return propagateErr
		}
		summary = propagated
		return nil
	})
	if err != nil {
		return model.ReceiverStation{}, ReintersectionSummary{}, err
	}
	return updated, summary, nil
}

// stationLocalizationChanged 判断站点更新是否影响交汇：
// 坐标、天线偏置、精度或启停用状态任一变化都要以当前值重新交汇
// （求解器只纳入 active 站点）；名称和校准时间变化不改变交汇输入。
func stationLocalizationChanged(before, after model.ReceiverStation) bool {
	return before.Latitude != after.Latitude ||
		before.Longitude != after.Longitude ||
		before.AntennaBiasDeg != after.AntennaBiasDeg ||
		before.AccuracyDeg != after.AccuracyDeg ||
		before.StationStatus != after.StationStatus
}

func (s *StationService) Coverage(ctx context.Context, id uint) (dto.StationCoverage, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return dto.StationCoverage{}, err
	}
	count, latest, err := s.repo.Coverage(ctx, id)
	if err != nil {
		return dto.StationCoverage{}, err
	}
	coverage := dto.StationCoverage{StationID: id, ObservationCount: count}
	if latest != nil {
		coverage.LastObservedAt = &latest.ObservedAt
	}
	return coverage, nil
}

func validateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return api.WithDetails(api.NewError(422, "INVALID_COORDINATES", "经纬度超出 WGS84 有效范围"), map[string]any{
			"latitude": latitude, "longitude": longitude,
		})
	}
	return nil
}
