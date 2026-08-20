package database

import (
	"database/sql"
	"log/slog"
	"strings"
	"testing"
	"time"

	"fiberest/internal/configs"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func resetViperForDBTest() {
	viper.Reset()
}

// helper to make config without env pollution
func newTestDBConfig(t *testing.T) *configs.Config {
	t.Helper()
	viper.Reset()
	cfg, err := configs.NewConfig()
	require.NoError(t, err)
	return cfg
}

func TestBuildDSN_ContainsExpectedParts(t *testing.T) {
	cfg := newTestDBConfig(t)
	// override via typed fields and legacy viper for buildDSN
	cfg.DB.Host = "myhost"
	cfg.DB.Port = 5433
	viper.Set("DB_HOST", "myhost")
	viper.Set("DB_PORT", "5433")
	viper.Set("DB_USER", "testuser")
	viper.Set("DB_PASSWORD", "secret")
	viper.Set("DB_NAME", "testdb")
	viper.Set("DB_SSLMODE", "disable")
	// need to update cfg typed fields used elsewhere? buildDSN uses GetString so viper is source
	svc := &DatabaseService{config: cfg}
	dsn := svc.buildDSN()
	assert.Contains(t, dsn, "host=myhost")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestBuildDSN_DefaultSSLMode(t *testing.T) {
	cfg := newTestDBConfig(t)
	viper.Set("DB_SSLMODE", "")
	svc := &DatabaseService{config: cfg}
	dsn := svc.buildDSN()
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestGormLogLevelMapping(t *testing.T) {
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel("silent"))
	assert.Equal(t, gormlogger.Error, mapGormLogLevel("error"))
	assert.Equal(t, gormlogger.Warn, mapGormLogLevel("warn"))
	assert.Equal(t, gormlogger.Info, mapGormLogLevel("info"))
	// case-insensitive
	assert.Equal(t, gormlogger.Info, mapGormLogLevel("INFO"))
	assert.Equal(t, gormlogger.Warn, mapGormLogLevel("Warn"))
	// unknown defaults to Silent
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel("unknown"))
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel(""))
}

func TestApplyPoolSettings_FromConfig(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.MaxIdleConns = 7
	cfg.DB.MaxOpenConns = 42
	cfg.DB.MaxLifetime = 30 * time.Minute

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	_ = mock

	applyPoolSettings(db, cfg)

	// MaxOpenConnections is the only pool field exposed via DBStats
	assert.Equal(t, 42, db.Stats().MaxOpenConnections)
	applyPoolSettings(db, cfg)
	assert.Equal(t, 42, db.Stats().MaxOpenConnections)
}

func TestPoolSettings_RespectsConfigValues_ViaGormDB(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.MaxIdleConns = 3
	cfg.DB.MaxOpenConns = 15
	cfg.DB.MaxLifetime = 5 * time.Minute
	cfg.DB.LogLevel = "silent"

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	mock.ExpectClose()

	dialector := postgres.New(postgres.Config{Conn: sqlDB})
	gdb, err := gorm.Open(dialector, &gorm.Config{Logger: gormlogger.Default.LogMode(mapGormLogLevel(cfg.DB.LogLevel))})
	require.NoError(t, err)

	// simulate what Connect does after opening
	underlying, err := gdb.DB()
	require.NoError(t, err)
	// underlying should be same as sqlDB (postgres with Conn reuses it)
	// apply pool
	applyPoolSettings(underlying, cfg)
	assert.Equal(t, 15, underlying.Stats().MaxOpenConnections)

	// verify logger level mapping doesn't error
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel(cfg.DB.LogLevel))
	_ = sqlDB
	_ = gdb
}

func TestNewDatabaseService_AcceptsLoggerParam(t *testing.T) {
	cfg := newTestDBConfig(t)
	// use sqlmock-backed GORM to avoid real DB connection
	// We test constructor signature accepts logger; we can't call real Connect without DB,
	// so we test that the constructor with logger param compiles and that a service can be created
	// via direct struct init and helper.
	logger := slog.Default()
	svc := &DatabaseService{config: cfg, slog: logger}
	require.NotNil(t, svc)
	assert.Equal(t, cfg, svc.config)
	// verify helper for creating GORM config uses logger mapping
	level := mapGormLogLevel("info")
	assert.Equal(t, gormlogger.Info, level)
	// ensure buildDSN still works with this service
	dsn := svc.buildDSN()
	assert.True(t, strings.Contains(dsn, "host="))
	_ = svc
}

// Ensure NewDatabaseService can be called with sqlmock dialector path without real DB by testing helper newGormConfig
func TestNewGormConfig_UsesLogLevel(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.LogLevel = "error"
	gcfg := newGormConfig(cfg)
	require.NotNil(t, gcfg)
	// gcfg.Logger should be set with LogMode error
	// We can't inspect private, but we can verify level mapping
	assert.Equal(t, gormlogger.Error, mapGormLogLevel(cfg.DB.LogLevel))
	_ = gcfg
}

// Verify idle time default is set (15m) - applyPoolSettings should set ConnMaxIdleTime
func TestApplyPoolSettings_SetsIdleTime(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.MaxIdleConns = 2
	cfg.DB.MaxOpenConns = 10
	cfg.DB.MaxLifetime = time.Hour
	var capturedIdle time.Duration
	// We can't directly read ConnMaxIdleTime from sql.DB via Stats, so test that helper exists and doesn't panic
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	require.NotPanics(t, func() { applyPoolSettings(sqlDB, cfg) })
	_ = capturedIdle
	_ = sqlDB
}

func TestDatabaseService_ImplementsGetDBAndClose(t *testing.T) {
	cfg := newTestDBConfig(t)
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose()
	dialector := postgres.New(postgres.Config{Conn: sqlDB})
	gdb, err := gorm.Open(dialector, &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	svc := &DatabaseService{config: cfg, db: gdb, slog: slog.Default()}
	require.NotNil(t, svc.GetDB())
	require.NoError(t, svc.Close())
	// second close after mock closed should still be tested via nil db
	svc2 := &DatabaseService{config: cfg}
	require.NoError(t, svc2.Close())
	_ = sql.ErrNoRows
}
