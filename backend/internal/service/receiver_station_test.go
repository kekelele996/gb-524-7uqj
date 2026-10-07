package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
)

func sanitizeDSNName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", "=", "_", "?", "_")
	return replacer.Replace(name)
}

type recalibrationHarness struct {
	db       *gorm.DB
	stations *StationService
	obsRepo  *repository.ObservationRepository
	caseRepo *repository.CaseRepository
	estRepo  *repository.EstimateRepository
	actor    repository.Actor
}

func newRecalibrationHarness(t *testing.T) recalibrationHarness {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+sanitizeDSNName(t.Name())+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&model.User{}, &model.ReceiverStation{}, &model.InterferenceCase{},
		&model.BearingObservation{}, &model.LocalizationEstimate{}, &model.AuditEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	stationRepo := repository.NewStationRepository(db)
	observationRepo := repository.NewObservationRepository(db)
	caseRepo := repository.NewCaseRepository(db)
	estimateRepo := repository.NewEstimateRepository(db)
	estimateService := NewEstimateService(estimateRepo, observationRepo, caseRepo, 1000)
	stationService := NewStationService(stationRepo, estimateService)
	return recalibrationHarness{
		db: db, stations: stationService, obsRepo: observationRepo,
		caseRepo: caseRepo, estRepo: estimateRepo,
		actor: repository.Actor{UserID: 7, Email: "analyst@spectrum.local", Role: constants.RoleAnalyst, RequestID: "test-request"},
	}
}

func (h recalibrationHarness) createStation(t *testing.T, code string, lat, lon, bias, accuracy float64, status string) model.ReceiverStation {
	t.Helper()
	calibrated := time.Now().UTC().Add(-time.Hour)
	station, err := h.stations.Create(context.Background(), dto.CreateStationRequest{
		StationCode: code, Name: "站点-" + code, Latitude: lat, Longitude: lon,
		AntennaBiasDeg: bias, AccuracyDeg: accuracy, StationStatus: status, CalibratedAt: &calibrated,
	}, h.actor)
	if err != nil {
		t.Fatalf("create station: %v", err)
	}
	return station
}

func (h recalibrationHarness) createCase(t *testing.T, code string, status constants.CaseStatus, frequency float64) model.InterferenceCase {
	t.Helper()
	item := model.InterferenceCase{
		CaseCode: code, Title: "案例-" + code, FrequencyCenterHz: frequency,
		CaseStatus: status, Priority: "normal", OpenedBy: h.actor.UserID, Version: 1,
	}
	if err := h.db.Create(&item).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}
	return item
}

func (h recalibrationHarness) addObservation(t *testing.T, stationID, caseID uint, bearing float64, quality constants.ObservationQuality, frequency, bandwidth float64) model.BearingObservation {
	t.Helper()
	corrected := normalizeBearing(bearing)
	observation := model.BearingObservation{
		StationID: stationID, CaseID: caseID, BearingDeg: bearing, CorrectedBearingDeg: corrected,
		SignalDBM: -70, FrequencyHz: frequency, BandwidthHz: bandwidth,
		ObservedAt: time.Now().UTC(), Quality: quality, CreatedBy: h.actor.UserID,
	}
	if err := h.db.Create(&observation).Error; err != nil {
		t.Fatalf("create observation: %v", err)
	}
	return observation
}

func updateStationRequest(station model.ReceiverStation, bias, accuracy float64, status string) dto.UpdateStationRequest {
	return dto.UpdateStationRequest{
		Name: station.Name, Latitude: station.Latitude, Longitude: station.Longitude,
		AntennaBiasDeg: bias, AccuracyDeg: accuracy, StationStatus: status, CalibratedAt: station.CalibratedAt,
	}
}

// Three stations with non-degenerate geometry around Shanghai; bearings point
// roughly toward (31.2304, 121.4737) so triangulation succeeds.
func seedReintersectableScene(t *testing.T, h recalibrationHarness, status constants.CaseStatus) (model.ReceiverStation, model.InterferenceCase) {
	t.Helper()
	s1 := h.createStation(t, "RX-A", 31.2504, 121.4437, 0, 1.2, "active")
	s2 := h.createStation(t, "RX-B", 31.2104, 121.4437, 0, 1.2, "active")
	s3 := h.createStation(t, "RX-C", 31.2304, 121.5137, 0, 1.2, "active")
	target := h.createCase(t, "CASE-OPEN", status, 433920000)
	h.addObservation(t, s1.ID, target.ID, 131, constants.QualityGood, 433920000, 25000)
	h.addObservation(t, s2.ID, target.ID, 41, constants.QualityGood, 433920000, 25000)
	h.addObservation(t, s3.ID, target.ID, 272, constants.QualityGood, 433920000, 25000)
	return s1, target
}

func TestStationCalibrationReintersectsOpenCase(t *testing.T) {
	h := newRecalibrationHarness(t)
	station, openCase := seedReintersectableScene(t, h, constants.CaseAnalyzing)

	before, err := h.obsRepo.Get(context.Background(), mustFirstObservationID(t, h, openCase.ID))
	if err != nil {
		t.Fatalf("load observation: %v", err)
	}
	updated, reintersection, err := h.stations.Update(context.Background(), station.ID,
		updateStationRequest(station, 2.5, station.AccuracyDeg, "active"), h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if !reintersection.Triggered || reintersection.Created != 1 || len(reintersection.OpenCases) != 1 {
		t.Fatalf("expected one reintersected case, got %+v", reintersection)
	}
	if reintersection.OpenCases[0].CaseID != openCase.ID || reintersection.OpenCases[0].EstimateID == 0 {
		t.Fatalf("unexpected reintersection summary: %+v", reintersection.OpenCases)
	}

	after, err := h.obsRepo.Get(context.Background(), before.ID)
	if err != nil {
		t.Fatalf("reload observation: %v", err)
	}
	if want := normalizeBearing(before.BearingDeg + updated.AntennaBiasDeg); after.CorrectedBearingDeg != want {
		t.Fatalf("corrected bearing = %v, want %v", after.CorrectedBearingDeg, want)
	}
	estimates, err := h.estRepo.List(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	if len(estimates) != 1 || estimates[0].EstimateStatus != constants.EstimateComplete {
		t.Fatalf("expected one appended complete estimate, got %+v", estimates)
	}
	reloadedCase, err := h.caseRepo.Get(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloadedCase.Version != openCase.Version+1 || reloadedCase.CaseStatus != constants.CaseAnalyzing {
		t.Fatalf("case version/status = %d/%s, want %d/analyzing", reloadedCase.Version, reloadedCase.CaseStatus, openCase.Version+1)
	}
}

func TestStationCalibrationLeavesFinalizedCaseUntouched(t *testing.T) {
	h := newRecalibrationHarness(t)
	station, finalizedCase := seedReintersectableScene(t, h, constants.CaseConfirmed)

	before, err := h.obsRepo.Get(context.Background(), mustFirstObservationID(t, h, finalizedCase.ID))
	if err != nil {
		t.Fatalf("load observation: %v", err)
	}
	_, reintersection, err := h.stations.Update(context.Background(), station.ID,
		updateStationRequest(station, 4.0, station.AccuracyDeg, "active"), h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	// All affected cases were finalized, so no re-intersection plan is built.
	if reintersection.Triggered || len(reintersection.OpenCases) != 0 || reintersection.Created != 0 || len(reintersection.Skipped) != 0 {
		t.Fatalf("finalized case must not appear in reintersection, got %+v", reintersection)
	}

	after, err := h.obsRepo.Get(context.Background(), before.ID)
	if err != nil {
		t.Fatalf("reload observation: %v", err)
	}
	if after.CorrectedBearingDeg != before.CorrectedBearingDeg {
		t.Fatalf("finalized observation bearing changed: %v -> %v", before.CorrectedBearingDeg, after.CorrectedBearingDeg)
	}
	estimates, err := h.estRepo.List(context.Background(), finalizedCase.ID)
	if err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	if len(estimates) != 0 {
		t.Fatalf("finalized case must not gain estimates, got %d", len(estimates))
	}
	reloadedCase, err := h.caseRepo.Get(context.Background(), finalizedCase.ID)
	if err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloadedCase.Version != finalizedCase.Version {
		t.Fatalf("finalized case version changed: %d -> %d", finalizedCase.Version, reloadedCase.Version)
	}
}

func TestStationCalibrationSkipsDegenerateOpenCase(t *testing.T) {
	h := newRecalibrationHarness(t)
	s1 := h.createStation(t, "RX-D1", 31.2004, 121.4737, 0, 1.2, "active")
	s2 := h.createStation(t, "RX-D2", 31.2204, 121.4737, 0, 1.2, "active")
	openCase := h.createCase(t, "CASE-DEGEN", constants.CaseAnalyzing, 433920000)
	// Both stations lie south of the source and emit the same bearing: parallel rays.
	h.addObservation(t, s1.ID, openCase.ID, 0, constants.QualityGood, 433920000, 25000)
	h.addObservation(t, s2.ID, openCase.ID, 0, constants.QualityGood, 433920000, 25000)

	_, reintersection, err := h.stations.Update(context.Background(), s1.ID,
		updateStationRequest(s1, 1.0, s1.AccuracyDeg, "active"), h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if len(reintersection.Skipped) != 1 || reintersection.Skipped[0].Reason != "GEOMETRY_DEGENERATE" {
		t.Fatalf("expected one degenerate skip, got %+v", reintersection.Skipped)
	}
	if reintersection.Created != 0 {
		t.Fatalf("degenerate geometry must not create estimates, got %d", reintersection.Created)
	}
	estimates, err := h.estRepo.List(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	if len(estimates) != 0 {
		t.Fatalf("degenerate case must not gain estimates, got %d", len(estimates))
	}
	// Bearings are still refreshed even when no new estimate can be formed.
	after, err := h.obsRepo.Get(context.Background(), mustFirstObservationID(t, h, openCase.ID))
	if err != nil {
		t.Fatalf("reload observation: %v", err)
	}
	if want := normalizeBearing(after.BearingDeg + 1.0); after.CorrectedBearingDeg != want {
		t.Fatalf("corrected bearing = %v, want %v", after.CorrectedBearingDeg, want)
	}
}

func TestStationBookkeepingEditDoesNotTriggerReintersection(t *testing.T) {
	h := newRecalibrationHarness(t)
	station, openCase := seedReintersectableScene(t, h, constants.CaseAnalyzing)
	request := updateStationRequest(station, station.AntennaBiasDeg, station.AccuracyDeg, "active")
	request.Name = "改名后的站点"
	_, reintersection, err := h.stations.Update(context.Background(), station.ID, request, h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if reintersection.Triggered {
		t.Fatalf("name-only edit must not trigger reintersection, got %+v", reintersection)
	}
	estimates, err := h.estRepo.List(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	if len(estimates) != 0 {
		t.Fatalf("name-only edit must not create estimates, got %d", len(estimates))
	}
}

func TestStationDisablementDropsStationFromOpenCaseReintersection(t *testing.T) {
	h := newRecalibrationHarness(t)
	station, openCase := seedReintersectableScene(t, h, constants.CaseAnalyzing)
	// Disabling one of three stations leaves two active stations, so the case
	// still re-intersects with the remaining geometry.
	_, reintersection, err := h.stations.Update(context.Background(), station.ID,
		updateStationRequest(station, station.AntennaBiasDeg, station.AccuracyDeg, "inactive"), h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if !reintersection.Triggered || reintersection.Created != 1 {
		t.Fatalf("expected a reintersection using the remaining stations, got %+v", reintersection)
	}
	estimates, err := h.estRepo.List(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	if len(estimates) != 1 || len(estimates[0].UsedObservationIDsJSON) == 0 {
		t.Fatalf("expected one estimate with used-observation evidence, got %+v", estimates)
	}
}

func TestStationCalibrationReintersectsOpenButNotFinalizedCase(t *testing.T) {
	h := newRecalibrationHarness(t)
	s1 := h.createStation(t, "RX-M1", 31.2504, 121.4437, 0, 1.2, "active")
	s2 := h.createStation(t, "RX-M2", 31.2104, 121.4437, 0, 1.2, "active")
	s3 := h.createStation(t, "RX-M3", 31.2304, 121.5137, 0, 1.2, "active")
	openCase := h.createCase(t, "CASE-MIX-OPEN", constants.CaseAnalyzing, 433920000)
	finalizedCase := h.createCase(t, "CASE-MIX-DONE", constants.CaseClosed, 433920000)
	for _, caseID := range []uint{openCase.ID, finalizedCase.ID} {
		h.addObservation(t, s1.ID, caseID, 131, constants.QualityGood, 433920000, 25000)
		h.addObservation(t, s2.ID, caseID, 41, constants.QualityGood, 433920000, 25000)
		h.addObservation(t, s3.ID, caseID, 272, constants.QualityGood, 433920000, 25000)
	}

	_, reintersection, err := h.stations.Update(context.Background(), s1.ID,
		updateStationRequest(s1, 3.0, s1.AccuracyDeg, "active"), h.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if !reintersection.Triggered || reintersection.Created != 1 || len(reintersection.OpenCases) != 1 {
		t.Fatalf("expected exactly one reintersected open case, got %+v", reintersection)
	}
	if reintersection.OpenCases[0].CaseID != openCase.ID {
		t.Fatalf("reintersected wrong case: %+v", reintersection.OpenCases)
	}

	openEstimates, err := h.estRepo.List(context.Background(), openCase.ID)
	if err != nil {
		t.Fatalf("list open estimates: %v", err)
	}
	if len(openEstimates) != 1 {
		t.Fatalf("open case should gain one estimate, got %d", len(openEstimates))
	}
	finalizedEstimates, err := h.estRepo.List(context.Background(), finalizedCase.ID)
	if err != nil {
		t.Fatalf("list finalized estimates: %v", err)
	}
	if len(finalizedEstimates) != 0 {
		t.Fatalf("closed case must not gain estimates, got %d", len(finalizedEstimates))
	}
	finalizedObservations, err := h.obsRepo.ListForCase(context.Background(), finalizedCase.ID, true)
	if err != nil {
		t.Fatalf("list finalized observations: %v", err)
	}
	for _, observation := range finalizedObservations {
		if observation.StationID == s1.ID && observation.CorrectedBearingDeg != normalizeBearing(observation.BearingDeg) {
			t.Fatalf("closed-case bearing was refreshed: obs %d corrected %v", observation.ID, observation.CorrectedBearingDeg)
		}
	}
	reloadedFinalized, err := h.caseRepo.Get(context.Background(), finalizedCase.ID)
	if err != nil {
		t.Fatalf("reload finalized case: %v", err)
	}
	if reloadedFinalized.Version != finalizedCase.Version || reloadedFinalized.CaseStatus != constants.CaseClosed {
		t.Fatalf("closed case changed: version %d status %s", reloadedFinalized.Version, reloadedFinalized.CaseStatus)
	}
}

func mustFirstObservationID(t *testing.T, h recalibrationHarness, caseID uint) uint {
	t.Helper()
	observations, err := h.obsRepo.ListForCase(context.Background(), caseID, true)
	if err != nil || len(observations) == 0 {
		t.Fatalf("load case observations: %v", err)
	}
	return observations[0].ID
}
