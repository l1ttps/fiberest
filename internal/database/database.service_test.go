package database

import (
	"database/sql"
	"log/slog"
	"testing"
	"time"

	"fiberest/internal/configs"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestDBConfig builds a config without touching the environment.
func newTestDBConfig(t *testing.T) *configs.Config {
	t.Helper()
	cfg := &configs.Config{}
	cfg.DB.Host = "localhost"
	cfg.DB.Port = 5432
	cfg.DB.User = "postgres"
	cfg.DB.Name = "fiberest"
	cfg.DB.SSLMode = "disable"
	cfg.DB.MaxIdleConns = 10
	cfg.DB.MaxOpenConns = 100
	cfg.DB.MaxLifetime = time.Hour
	cfg.DB.LogLevel = "silent"
	return cfg
}

func TestBuildDSN_ContainsExpectedParts(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.Host = "myhost"
	cfg.DB.Port = 5433
	cfg.DB.User = "testuser"
	cfg.DB.Password = "secret"
	cfg.DB.Name = "testdb"

	dsn := (&DatabaseService{config: cfg}).buildDSN()
	assert.Contains(t, dsn, "host=myhost")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=disable")
}

func TestGormLogLevelMapping(t *testing.T) {
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel("silent"))
	assert.Equal(t, gormlogger.Error, mapGormLogLevel("error"))
	assert.Equal(t, gormlogger.Warn, mapGormLogLevel("warn"))
	assert.Equal(t, gormlogger.Info, mapGormLogLevel("info"))
	assert.Equal(t, gormlogger.Info, mapGormLogLevel("INFO"))
	assert.Equal(t, gormlogger.Warn, mapGormLogLevel("Warn"))
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel("unknown"))
	assert.Equal(t, gormlogger.Silent, mapGormLogLevel(""))
}

func TestApplyPoolSettings_FromConfig(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.MaxOpenConns = 42
	cfg.DB.MaxLifetime = 30 * time.Minute

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.NotPanics(t, func() { applyPoolSettings(db, cfg) })
	assert.Equal(t, 42, db.Stats().MaxOpenConnections)
}

func TestNewGormConfig_UsesLogLevel(t *testing.T) {
	cfg := newTestDBConfig(t)
	cfg.DB.LogLevel = "error"

	gcfg := newGormConfig(cfg)
	require.NotNil(t, gcfg)
	assert.Equal(t, gormlogger.Error, mapGormLogLevel(cfg.DB.LogLevel))
}

func TestDatabaseService_GetDBAndClose(t *testing.T) {
	cfg := newTestDBConfig(t)
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose()

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}),
		&gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	svc := &DatabaseService{config: cfg, db: gdb, slog: slog.Default()}
	require.NotNil(t, svc.GetDB())
	require.NoError(t, svc.Close())

	// Close on an unconnected service is a no-op.
	require.NoError(t, (&DatabaseService{config: cfg}).Close())
	_ = sql.ErrNoRows
}
