package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fiberest/internal/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DatabaseService provides database connection management using GORM
type DatabaseService struct {
	db     *gorm.DB
	config *configs.Config
	slog   *slog.Logger
}

// NewDatabaseService creates a new DatabaseService instance (config-driven).
// Logger is optional — when nil, slog.Default() is used to avoid fx cycles
// and to keep direct construction simple in tests.
func NewDatabaseService(config *configs.Config, logger *slog.Logger) (*DatabaseService, error) {
	if logger == nil {
		logger = slog.Default()
	}
	service := &DatabaseService{
		config: config,
		slog:   logger,
	}

	if err := service.Connect(); err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	return service, nil
}

// Connect establishes connection to PostgreSQL database
func (s *DatabaseService) Connect() error {
	dsn := s.buildDSN()

	gcfg := newGormConfig(s.config)

	db, err := gorm.Open(postgres.Open(dsn), gcfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	applyPoolSettings(sqlDB, s.config)

	s.db = db
	return nil
}

// newGormConfig builds a gorm.Config with log level derived from cfg.DB.LogLevel.
func newGormConfig(cfg *configs.Config) *gorm.Config {
	level := mapGormLogLevel(cfg.DB.LogLevel)
	return &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
	}
}

// mapGormLogLevel maps DB_LOG_LEVEL string to gorm logger level.
// silent→Silent, error→Error, warn→Warn, info→Info; unknown → Silent.
// Case-insensitive.
func mapGormLogLevel(s string) gormlogger.LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn":
		return gormlogger.Warn
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Silent
	}
}

// applyPoolSettings configures the sql.DB pool from typed Config.
// MaxLifetime comes from cfg.DB.MaxLifetime; idle time uses a 15m default.
//
// ponytail: 15m ConnMaxIdleTime is a fixed operational default, not yet config-driven;
// upgrade path is adding DB_CONN_MAX_IDLE_TIME to Config and reading it here when needed.
func applyPoolSettings(sqlDB *sql.DB, cfg *configs.Config) {
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.DB.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)
}

// buildDSN constructs PostgreSQL connection string from configuration
func (s *DatabaseService) buildDSN() string {
	host := s.config.GetString("DB_HOST")
	port := s.config.GetString("DB_PORT")
	user := s.config.GetString("DB_USER")
	password := s.config.GetString("DB_PASSWORD")
	dbname := s.config.GetString("DB_NAME")

	sslmode := s.config.GetString("DB_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}

// GetDB returns the GORM database instance
func (s *DatabaseService) GetDB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *DatabaseService) Close() error {
	if s.db == nil {
		return nil
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}

// Ping checks if the database connection is alive
func (s *DatabaseService) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
