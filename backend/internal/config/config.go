package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"spectrum-interference-triangulation/backend/internal/constants"
	"spectrum-interference-triangulation/backend/internal/model"
)

type Config struct {
	Port                   string
	DBDriver               string
	DBDSN                  string
	DBHost                 string
	DBPort                 string
	DBName                 string
	DBUser                 string
	DBPassword             string
	DBSSLMode              string
	DBAutoMigrate          bool
	SeedData               bool
	JWTSecret              string
	CORSOrigins            []string
	GeometryConditionLimit float64
	LogLevel               slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		Port:                   env("PORT", "8080"),
		DBDriver:               strings.ToLower(env("DB_DRIVER", "postgres")),
		DBDSN:                  os.Getenv("DB_DSN"),
		DBHost:                 env("DB_HOST", "127.0.0.1"),
		DBPort:                 env("DB_INTERNAL_PORT", "5432"),
		DBName:                 env("DB_NAME", "spectrum_localization"),
		DBUser:                 env("DB_USER", "spectrum_app"),
		DBPassword:             env("DB_PASSWORD", "spectrum-local-dev-password"),
		DBSSLMode:              env("DB_SSLMODE", "disable"),
		DBAutoMigrate:          envBool("DB_AUTO_MIGRATE", true),
		SeedData:               envBool("SEED_DATA", true),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		CORSOrigins:            splitCSV(env("CORS_ORIGINS", "http://127.0.0.1:18524,http://localhost:18524")),
		GeometryConditionLimit: envFloat("GEOMETRY_CONDITION_LIMIT", 1000),
		LogLevel:               parseLogLevel(env("LOG_LEVEL", "info")),
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}
	if cfg.DBDriver == "sqlite" && cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required for sqlite")
	}
	if cfg.GeometryConditionLimit < 10 {
		return Config{}, errors.New("GEOMETRY_CONDITION_LIMIT must be at least 10")
	}
	return cfg, nil
}

func OpenDatabase(cfg Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBDriver == "sqlite" {
		dialector = sqlite.Open(cfg.DBDSN)
	} else {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)
		dialector = postgres.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	if cfg.DBAutoMigrate {
		if err := migrate(db); err != nil {
			return nil, fmt.Errorf("migrate database: %w", err)
		}
	}
	if cfg.SeedData {
		if err := seed(db); err != nil {
			return nil, fmt.Errorf("seed database: %w", err)
		}
	}
	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.ReceiverStation{},
		&model.InterferenceCase{},
		&model.BearingObservation{},
		&model.LocalizationEstimate{},
		&model.AuditEvent{},
	)
}

func seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("Spectrum!2026"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash seed password: %w", err)
		}
		users := []model.User{
			{Email: "observer@spectrum.local", DisplayName: "测向观测员", PasswordHash: string(passwordHash), Role: constants.RoleObserver, Active: true},
			{Email: "analyst@spectrum.local", DisplayName: "频谱分析员", PasswordHash: string(passwordHash), Role: constants.RoleAnalyst, Active: true},
			{Email: "reviewer@spectrum.local", DisplayName: "定位复核员", PasswordHash: string(passwordHash), Role: constants.RoleReviewer, Active: true},
			{Email: "admin@spectrum.local", DisplayName: "系统管理员", PasswordHash: string(passwordHash), Role: constants.RoleAdmin, Active: true},
		}
		if err := tx.Create(&users).Error; err != nil {
			return err
		}
		calibrated := time.Now().UTC().Add(-72 * time.Hour)
		stations := []model.ReceiverStation{
			{StationCode: "RX-WEST", Name: "西侧固定测向站", Latitude: 31.2304, Longitude: 121.4437, AccuracyDeg: 1.2, AntennaBiasDeg: 0.4, StationStatus: "active", CalibratedAt: &calibrated},
			{StationCode: "RX-SOUTH", Name: "南侧移动测向站", Latitude: 31.2104, Longitude: 121.4737, AccuracyDeg: 1.5, AntennaBiasDeg: -0.3, StationStatus: "active", CalibratedAt: &calibrated},
			{StationCode: "RX-EAST", Name: "东侧固定测向站", Latitude: 31.2304, Longitude: 121.5037, AccuracyDeg: 1.0, AntennaBiasDeg: 0.1, StationStatus: "active", CalibratedAt: &calibrated},
			{StationCode: "RX-NORTH", Name: "北侧验证测向站", Latitude: 31.2504, Longitude: 121.4737, AccuracyDeg: 1.8, AntennaBiasDeg: 0, StationStatus: "active", CalibratedAt: &calibrated},
		}
		if err := tx.Create(&stations).Error; err != nil {
			return err
		}
		cases := []model.InterferenceCase{
			{CaseCode: "RF-2026-001", Title: "433.920 MHz 间歇窄带信号", FrequencyCenterHz: 433920000, CaseStatus: constants.CaseAnalyzing, Priority: "high", OpenedBy: users[1].ID, Version: 3},
			{CaseCode: "RF-2026-002", Title: "868.300 MHz 复核案例", FrequencyCenterHz: 868300000, CaseStatus: constants.CasePendingReview, Priority: "normal", OpenedBy: users[1].ID, Version: 4, Conclusion: "三站交汇稳定，等待独立复核。"},
		}
		if err := tx.Create(&cases).Error; err != nil {
			return err
		}
		now := time.Now().UTC().Add(-20 * time.Minute)
		observations := []model.BearingObservation{
			{StationID: stations[0].ID, CaseID: cases[0].ID, BearingDeg: 89.6, CorrectedBearingDeg: 90.0, SignalDBM: -67, FrequencyHz: 433920300, BandwidthHz: 12500, ObservedAt: now, Quality: constants.QualityGood, CreatedBy: users[0].ID},
			{StationID: stations[1].ID, CaseID: cases[0].ID, BearingDeg: 0.5, CorrectedBearingDeg: 0.2, SignalDBM: -71, FrequencyHz: 433919700, BandwidthHz: 12500, ObservedAt: now.Add(time.Minute), Quality: constants.QualityGood, CreatedBy: users[0].ID},
			{StationID: stations[2].ID, CaseID: cases[0].ID, BearingDeg: 269.7, CorrectedBearingDeg: 269.8, SignalDBM: -64, FrequencyHz: 433920100, BandwidthHz: 12500, ObservedAt: now.Add(2 * time.Minute), Quality: constants.QualityFair, CreatedBy: users[0].ID},
			{StationID: stations[3].ID, CaseID: cases[0].ID, BearingDeg: 205, CorrectedBearingDeg: 205, SignalDBM: -82, FrequencyHz: 433920500, BandwidthHz: 12500, ObservedAt: now.Add(3 * time.Minute), Quality: constants.QualityPoor, CreatedBy: users[0].ID},
			{StationID: stations[0].ID, CaseID: cases[1].ID, BearingDeg: 92, CorrectedBearingDeg: 92.4, SignalDBM: -73, FrequencyHz: 868300000, BandwidthHz: 25000, ObservedAt: now, Quality: constants.QualityGood, CreatedBy: users[0].ID},
		}
		if err := tx.Create(&observations).Error; err != nil {
			return err
		}
		return tx.Create(&model.AuditEvent{
			RequestID: "seed-bootstrap", UserID: users[3].ID, ActorEmail: users[3].Email,
			Action: "system.seeded", EntityType: "system", EntityID: 1,
			BeforeJSON: "{}", AfterJSON: `{"stations":4,"cases":2,"observations":5}`,
			CreatedAt: time.Now().UTC(),
		}).Error
	})
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
