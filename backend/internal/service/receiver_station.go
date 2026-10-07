package service

import (
	"context"
	"strings"
	"time"

	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
	"spectrum-interference-triangulation/backend/pkg/api"
)

type StationService struct {
	repo *repository.StationRepository
}

func NewStationService(repo *repository.StationRepository) *StationService {
	return &StationService{repo: repo}
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
		return model.ReceiverStation{}, api.NewError(422, "CALIBRATION_REQUIRED", "启用测向站前必须填写最近校准时间")
	}
	if err := s.repo.Create(ctx, &station, actor); err != nil {
		return model.ReceiverStation{}, err
	}
	return station, nil
}

func (s *StationService) Update(ctx context.Context, id uint, request dto.UpdateStationRequest, actor repository.Actor) (model.ReceiverStation, error) {
	before, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.ReceiverStation{}, err
	}
	if err := validateCoordinates(request.Latitude, request.Longitude); err != nil {
		return model.ReceiverStation{}, err
	}
	if request.CalibratedAt != nil && request.CalibratedAt.After(time.Now().UTC().Add(5*time.Minute)) {
		return model.ReceiverStation{}, api.NewError(422, "INVALID_CALIBRATION_TIME", "校准时间不能晚于当前时间")
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
		return model.ReceiverStation{}, api.NewError(422, "CALIBRATION_REQUIRED", "启用测向站前必须填写最近校准时间")
	}
	if err := s.repo.Update(ctx, &updated, before, actor); err != nil {
		return model.ReceiverStation{}, err
	}
	return updated, nil
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
