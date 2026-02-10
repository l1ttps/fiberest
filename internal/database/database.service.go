package database

import (
	"context"
	"fmt"
	"time"

	"fiberest/internal/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DatabaseService provides database connection management using GORM
type DatabaseService struct {
	db     *gorm.DB
	config *configs.Config
}

// NewDatabaseService creates a new DatabaseService instance
func NewDatabaseService(config *configs.Config) (*DatabaseService, error) {
	service := &DatabaseService{
		config: config,
	}

	if err := service.Connect(); err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	return service, nil
}

// Connect establishes connection to PostgreSQL database
func (s *DatabaseService) Connect() error {
	dsn := s.buildDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	s.db = db
	return nil
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
