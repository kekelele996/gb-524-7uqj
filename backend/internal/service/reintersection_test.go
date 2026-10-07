package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/dto"
	"spectrum-interference-triangulation/backend/internal/model"
	"spectrum-interference-triangulation/backend/internal/repository"
)

type reintersectionFixture struct {
	db             *gorm.DB
	stationSvc     *StationService
	estimateSvc    *EstimateService
	caseSvc        *CaseService
	observationSvc *ObservationService
	stations       []model.ReceiverStation
	cases          []model.InterferenceCase
	actor          repository.Actor
}

func newReintersectionFixture(t *testing.T) *reintersectionFixture {
	t.Helper()
	dsn := fmt.Sprintf("file:reintersection-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.ReceiverStation{}, &model.InterferenceCase{},
		&model.BearingObservation{}, &model.LocalizationEstimate{}, &model.AuditEvent{},
	); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}

	stationRepo := repository.NewStationRepository(db)
	observationRepo := repository.NewObservationRepository(db)
	caseRepo := repository.NewCaseRepository(db)
	estimateRepo := repository.NewEstimateRepository(db)
	txManager := repository.NewTransactionManager(db)

	estimateSvc := NewEstimateService(estimateRepo, observationRepo, caseRepo, 1000)
	fixture := &reintersectionFixture{
		db:             db,
		stationSvc:     NewStationService(stationRepo, txManager, estimateSvc),
		estimateSvc:    estimateSvc,
		caseSvc:        NewCaseService(caseRepo),
		observationSvc: NewObservationService(observationRepo, stationRepo, caseRepo),
		actor:          repository.Actor{UserID: 7, Email: "analyst@spectrum.local", Role: constants.RoleAnalyst, RequestID: "test-request"},
	}
	return fixture
}

func (f *reintersectionFixture) seedAnalyzingCase(t *testing.T, status constants.CaseStatus) (model.ReceiverStation, model.InterferenceCase) {
	t.Helper()
	calibrated := time.Now().UTC().Add(-time.Hour)
	// 三个启用站分布在中心的西、南、东，方位近似指向中心，保证几何不退化。
	stations := []model.ReceiverStation{
		{StationCode: "ST-W", Name: "西站", Latitude: 31.2304, Longitude: 121.4437, AccuracyDeg: 1.0, AntennaBiasDeg: 0, StationStatus: "active", CalibratedAt: &calibrated},
		{StationCode: "ST-S", Name: "南站", Latitude: 31.2104, Longitude: 121.4737, AccuracyDeg: 1.0, AntennaBiasDeg: 0, StationStatus: "active", CalibratedAt: &calibrated},
		{StationCode: "ST-E", Name: "东站", Latitude: 31.2304, Longitude: 121.5037, AccuracyDeg: 1.0, AntennaBiasDeg: 0, StationStatus: "active", CalibratedAt: &calibrated},
	}
	for i := range stations {
		if err := f.db.Create(&stations[i]).Error; err != nil {
			t.Fatalf("create station: %v", err)
		}
	}
	caseRecord := model.InterferenceCase{
		CaseCode: "CASE-OPEN", Title: "未结案案例", FrequencyCenterHz: 433920000,
		CaseStatus: constants.CaseAnalyzing, Priority: "normal", OpenedBy: f.actor.UserID, Version: 1,
	}
	if err := f.db.Create(&caseRecord).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}
	if status != constants.CaseAnalyzing {
		if err := f.db.Model(&caseRecord).Update("case_status", string(status)).Error; err != nil {
			t.Fatalf("set case status: %v", err)
		}
		caseRecord.CaseStatus = status
	}
	f.stations = stations
	f.cases = []model.InterferenceCase{caseRecord}
	return stations[0], caseRecord
}

func (f *reintersectionFixture) seedObservations(t *testing.T, caseID uint, bearings []float64) {
	t.Helper()
	observedAt := time.Now().UTC().Add(-30 * time.Minute)
	for i, bearing := range bearings {
		observation := model.BearingObservation{
			StationID: f.stations[i].ID, CaseID: caseID,
			BearingDeg: bearing, CorrectedBearingDeg: bearing,
			SignalDBM: -70, FrequencyHz: 433920000, BandwidthHz: 25000,
			ObservedAt: observedAt.Add(time.Duration(i) * time.Minute),
			Quality:    constants.QualityGood, CreatedBy: f.actor.UserID,
		}
		if err := f.db.Create(&observation).Error; err != nil {
			t.Fatalf("create observation: %v", err)
		}
	}
}

func (f *reintersectionFixture) runLocalization(t *testing.T, caseID uint) {
	t.Helper()
	_, err := f.estimateSvc.Run(context.Background(), dto.RunLocalizationRequest{CaseID: caseID, AllowOutlier: false}, f.actor)
	if err != nil {
		t.Fatalf("seed localization run: %v", err)
	}
}

func estimateIDs(t *testing.T, db *gorm.DB, caseID uint) []model.LocalizationEstimate {
	t.Helper()
	var estimates []model.LocalizationEstimate
	if err := db.Where("case_id = ?", caseID).Order("id ASC").Find(&estimates).Error; err != nil {
		t.Fatalf("list estimates: %v", err)
	}
	return estimates
}

func auditActions(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var events []model.AuditEvent
	if err := db.Order("id ASC").Find(&events).Error; err != nil {
		t.Fatalf("list audits: %v", err)
	}
	actions := make([]string, 0, len(events))
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	return actions
}

// 场景一：analyzing 案例已有定位。站点偏置变化后，观测校正方位被重算，
// 并追加一条以站点当前值交汇的新估计，旧估计保持不变，案例 version 递增。
func TestStationUpdateReintersectsOpenAnalyzingCase(t *testing.T) {
	fixture := newReintersectionFixture(t)
	ctx := context.Background()
	station, caseRecord := fixture.seedAnalyzingCase(t, constants.CaseAnalyzing)
	// 西站在中心正西，射线指向正东约 89°；南站正北约 2°；东站指向正西约 271°。
	fixture.seedObservations(t, caseRecord.ID, []float64{89, 2, 271})
	fixture.runLocalization(t, caseRecord.ID)
	if err := fixture.db.First(&caseRecord, caseRecord.ID).Error; err != nil {
		t.Fatalf("reload seeded case: %v", err)
	}

	beforeEstimates := estimateIDs(t, fixture.db, caseRecord.ID)
	if len(beforeEstimates) != 1 {
		t.Fatalf("expected 1 seed estimate, got %d", len(beforeEstimates))
	}

	update := dto.UpdateStationRequest{
		Name: station.Name, Latitude: station.Latitude, Longitude: station.Longitude,
		AntennaBiasDeg: 1.5, AccuracyDeg: station.AccuracyDeg,
		StationStatus: station.StationStatus, CalibratedAt: station.CalibratedAt,
	}
	updated, summary, err := fixture.stationSvc.Update(ctx, station.ID, update, fixture.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if updated.AntennaBiasDeg != 1.5 {
		t.Fatalf("antenna bias not persisted: %v", updated.AntennaBiasDeg)
	}
	if len(summary.ReintersectedCaseIDs) != 1 || summary.ReintersectedCaseIDs[0] != caseRecord.ID {
		t.Fatalf("expected case %d reintersected, got %+v", caseRecord.ID, summary)
	}
	if summary.RecalibratedObservationCount != 1 {
		t.Fatalf("expected 1 recalibrated observation, got %d", summary.RecalibratedObservationCount)
	}

	var observation model.BearingObservation
	if err := fixture.db.Where("station_id = ? AND case_id = ?", station.ID, caseRecord.ID).First(&observation).Error; err != nil {
		t.Fatalf("reload observation: %v", err)
	}
	if observation.BearingDeg != 89 {
		t.Fatalf("raw bearing must be preserved, got %v", observation.BearingDeg)
	}
	if observation.CorrectedBearingDeg != 90.5 {
		t.Fatalf("corrected bearing should be recomputed to 90.5, got %v", observation.CorrectedBearingDeg)
	}

	afterEstimates := estimateIDs(t, fixture.db, caseRecord.ID)
	if len(afterEstimates) != 2 {
		t.Fatalf("expected appended reintersection estimate, got %d", len(afterEstimates))
	}
	if afterEstimates[0].ID != beforeEstimates[0].ID || afterEstimates[0].Latitude != beforeEstimates[0].Latitude {
		t.Fatal("previous estimate must remain immutable")
	}
	if afterEstimates[1].ID == beforeEstimates[0].ID {
		t.Fatal("reintersection must create a new estimate row")
	}
	if afterEstimates[1].CreatedBy != fixture.actor.UserID {
		t.Fatalf("reintersection estimate actor = %d, want %d", afterEstimates[1].CreatedBy, fixture.actor.UserID)
	}
	snapshot := afterEstimates[1].InputSnapshotJSON.String()
	if !json.Valid([]byte(snapshot)) {
		t.Fatalf("reintersection snapshot must be valid JSON: %s", snapshot)
	}

	var reloadedCase model.InterferenceCase
	if err := fixture.db.First(&reloadedCase, caseRecord.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloadedCase.Version != caseRecord.Version+1 {
		t.Fatalf("case version = %d, want %d", reloadedCase.Version, caseRecord.Version+1)
	}

	actions := auditActions(t, fixture.db)
	assertContainsAction(t, actions, "bearing_observation.recalibrated")
	assertContainsAction(t, actions, "localization_estimate.reintersected")
}

// 场景二：confirmed 与 closed 案例的观测和定位在站点变化后完全冻结。
func TestStationUpdateKeepsFinalizedCasesFrozen(t *testing.T) {
	for _, status := range []constants.CaseStatus{constants.CaseConfirmed, constants.CaseClosed} {
		t.Run(string(status), func(t *testing.T) {
			fixture := newReintersectionFixture(t)
			ctx := context.Background()
			station, caseRecord := fixture.seedAnalyzingCase(t, status)
			fixture.seedObservations(t, caseRecord.ID, []float64{89, 2, 271})

			// 直接插入一条"历史"定位，模拟确认/关闭前跑过的交汇。
			historical := model.LocalizationEstimate{
				CaseID: caseRecord.ID, AlgorithmVersion: "wls-bearing-v1",
				Latitude: 31.23, Longitude: 121.47, UncertaintyRadiusM: 1000,
				ResidualDeg: 0.5, ConditionNumber: 4, GeometryDegenerate: false,
				UsedObservationIDsJSON: mustJSON(t, []uint{1, 2, 3}), OutlierIDsJSON: mustJSON(t, []uint{}),
				ResidualsJSON: mustJSON(t, []any{}), InputSnapshotJSON: mustJSON(t, []any{}),
				EstimateStatus: constants.EstimateComplete, CreatedBy: fixture.actor.UserID,
			}
			if err := fixture.db.Create(&historical).Error; err != nil {
				t.Fatalf("create historical estimate: %v", err)
			}

			update := dto.UpdateStationRequest{
				Name: station.Name, Latitude: station.Latitude + 0.01, Longitude: station.Longitude,
				AntennaBiasDeg: 2.0, AccuracyDeg: 2.5,
				StationStatus: station.StationStatus, CalibratedAt: station.CalibratedAt,
			}
			_, summary, err := fixture.stationSvc.Update(ctx, station.ID, update, fixture.actor)
			if err != nil {
				t.Fatalf("update station: %v", err)
			}
			if len(summary.AffectedCaseIDs) != 0 || len(summary.ReintersectedCaseIDs) != 0 {
				t.Fatalf("finalized case must not be touched, got %+v", summary)
			}

			var observation model.BearingObservation
			if err := fixture.db.Where("station_id = ?", station.ID).First(&observation).Error; err != nil {
				t.Fatalf("load observation: %v", err)
			}
			if observation.CorrectedBearingDeg != 89 {
				t.Fatalf("finalized observation corrected bearing frozen at 89, got %v", observation.CorrectedBearingDeg)
			}
			estimates := estimateIDs(t, fixture.db, caseRecord.ID)
			if len(estimates) != 1 || estimates[0].ID != historical.ID {
				t.Fatalf("finalized case estimates must be untouched, got %d rows", len(estimates))
			}
			var reloadedCase model.InterferenceCase
			if err := fixture.db.First(&reloadedCase, caseRecord.ID).Error; err != nil {
				t.Fatalf("reload case: %v", err)
			}
			if reloadedCase.Version != caseRecord.Version {
				t.Fatalf("finalized case version must not advance: %d -> %d", caseRecord.Version, reloadedCase.Version)
			}
			for _, action := range auditActions(t, fixture.db) {
				if action == "bearing_observation.recalibrated" || action == "localization_estimate.reintersected" {
					t.Fatalf("finalized case generated %s audit", action)
				}
			}
		})
	}
}

// 场景三：从未跑过定位的未结案案例只重算校正方位，不自动补跑交汇，也不推进版本。
func TestStationUpdateSkipsCaseWithoutPriorEstimate(t *testing.T) {
	fixture := newReintersectionFixture(t)
	ctx := context.Background()
	station, caseRecord := fixture.seedAnalyzingCase(t, constants.CaseCollecting)
	fixture.seedObservations(t, caseRecord.ID, []float64{89, 2, 271})

	update := dto.UpdateStationRequest{
		Name: station.Name, Latitude: station.Latitude, Longitude: station.Longitude,
		AntennaBiasDeg: -1.0, AccuracyDeg: station.AccuracyDeg,
		StationStatus: station.StationStatus, CalibratedAt: station.CalibratedAt,
	}
	_, summary, err := fixture.stationSvc.Update(ctx, station.ID, update, fixture.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if len(summary.ReintersectedCaseIDs) != 0 {
		t.Fatalf("case without prior estimate must not be reintersected: %+v", summary)
	}
	if len(summary.SkippedCases) != 1 || summary.SkippedCases[0].Reason != skipNoPriorEstimate {
		t.Fatalf("expected no_prior_estimate skip, got %+v", summary.SkippedCases)
	}
	if summary.RecalibratedObservationCount != 1 {
		t.Fatalf("corrected bearing must still be recomputed, got %d", summary.RecalibratedObservationCount)
	}
	if len(estimateIDs(t, fixture.db, caseRecord.ID)) != 0 {
		t.Fatal("no estimate should be appended for case without prior run")
	}
	var reloadedCase model.InterferenceCase
	if err := fixture.db.First(&reloadedCase, caseRecord.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloadedCase.Version != caseRecord.Version {
		t.Fatalf("case without estimate must not bump version: %d", reloadedCase.Version)
	}
}

// 场景四：只改名称或校准时间不触碰观测与定位；坐标或精度变化同样触发交汇。
func TestStationUpdateIgnoresNonGeometryChanges(t *testing.T) {
	fixture := newReintersectionFixture(t)
	ctx := context.Background()
	station, caseRecord := fixture.seedAnalyzingCase(t, constants.CaseAnalyzing)
	fixture.seedObservations(t, caseRecord.ID, []float64{89, 2, 271})
	fixture.runLocalization(t, caseRecord.ID)

	calibratedAt := time.Now().UTC().Add(-time.Minute)
	update := dto.UpdateStationRequest{
		Name: "西站-改名", Latitude: station.Latitude, Longitude: station.Longitude,
		AntennaBiasDeg: station.AntennaBiasDeg, AccuracyDeg: station.AccuracyDeg,
		StationStatus: station.StationStatus, CalibratedAt: &calibratedAt,
	}
	_, summary, err := fixture.stationSvc.Update(ctx, station.ID, update, fixture.actor)
	if err != nil {
		t.Fatalf("update station: %v", err)
	}
	if summary.LocalizationChanged {
		t.Fatal("name/calibration change must not trigger reintersection")
	}
	if len(summary.AffectedCaseIDs) != 0 || summary.RecalibratedObservationCount != 0 {
		t.Fatalf("non-localization update must not touch cases: %+v", summary)
	}
	if len(estimateIDs(t, fixture.db, caseRecord.ID)) != 1 {
		t.Fatal("non-localization update must not append estimates")
	}
}

// 场景五：站点停用会让该站观测退出交汇，未结案（含 pending_review）案例仍需以当前值重新交汇；
// 校正方位不随停用重算，变化仅体现在新估计的输入快照中。
func TestStationDeactivationReintersectsPendingReviewCase(t *testing.T) {
	fixture := newReintersectionFixture(t)
	ctx := context.Background()
	station, caseRecord := fixture.seedAnalyzingCase(t, constants.CasePendingReview)
	fixture.seedObservations(t, caseRecord.ID, []float64{89, 2, 271})
	// 用 analyzing 状态跑出历史估计后再置回 pending_review。
	if err := fixture.db.Model(&caseRecord).Update("case_status", string(constants.CaseAnalyzing)).Error; err != nil {
		t.Fatalf("move case to analyzing: %v", err)
	}
	fixture.runLocalization(t, caseRecord.ID)
	if err := fixture.db.Model(&caseRecord).Update("case_status", string(constants.CasePendingReview)).Error; err != nil {
		t.Fatalf("move case to pending_review: %v", err)
	}
	if err := fixture.db.First(&caseRecord, caseRecord.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	historical := estimateIDs(t, fixture.db, caseRecord.ID)
	if len(historical) != 1 {
		t.Fatalf("expected 1 historical estimate, got %d", len(historical))
	}

	calibratedAt := station.CalibratedAt
	update := dto.UpdateStationRequest{
		Name: station.Name, Latitude: station.Latitude, Longitude: station.Longitude,
		AntennaBiasDeg: station.AntennaBiasDeg, AccuracyDeg: station.AccuracyDeg,
		StationStatus: "inactive", CalibratedAt: calibratedAt,
	}
	_, summary, err := fixture.stationSvc.Update(ctx, station.ID, update, fixture.actor)
	if err != nil {
		t.Fatalf("deactivate station: %v", err)
	}
	if !summary.LocalizationChanged || len(summary.ReintersectedCaseIDs) != 1 {
		t.Fatalf("pending_review case must be reintersected on deactivation, got %+v", summary)
	}
	if summary.RecalibratedObservationCount != 0 {
		t.Fatalf("deactivation must not recompute corrected bearings, got %d", summary.RecalibratedObservationCount)
	}

	estimates := estimateIDs(t, fixture.db, caseRecord.ID)
	if len(estimates) != 2 {
		t.Fatalf("expected appended reintersection, got %d", len(estimates))
	}
	var inputs []map[string]any
	if err := json.Unmarshal(estimates[1].InputSnapshotJSON, &inputs); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("deactivated station must drop out of reintersection, got %d inputs", len(inputs))
	}
	for _, input := range inputs {
		if input["station_code"] == station.StationCode {
			t.Fatal("deactivated station still participates in reintersection")
		}
	}
	var reloadedCase model.InterferenceCase
	if err := fixture.db.First(&reloadedCase, caseRecord.ID).Error; err != nil {
		t.Fatalf("reload case: %v", err)
	}
	if reloadedCase.CaseStatus != constants.CasePendingReview {
		t.Fatalf("reintersection must not change case status, got %s", reloadedCase.CaseStatus)
	}
	if reloadedCase.Version != caseRecord.Version+1 {
		t.Fatalf("pending_review case version = %d, want %d", reloadedCase.Version, caseRecord.Version+1)
	}
}

func assertContainsAction(t *testing.T, actions []string, want string) {
	t.Helper()
	for _, action := range actions {
		if action == want {
			return
		}
	}
	t.Fatalf("actions %v do not contain %q", actions, want)
}

func mustJSON(t *testing.T, value any) datatypes.JSON {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return datatypes.JSON(data)
}
