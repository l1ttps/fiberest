package database

import (
	"context"
	"log/slog"

	"fiberest/internal/configs"
	"fiberest/internal/models"

	"go.uber.org/fx"
)

// Module provides database-related dependencies.
//
// Schema changes live here and only here: one AutoMigrate for every model,
// gated by DB_AUTO_MIGRATE (off by default in production).
var Module = fx.Options(
	fx.Provide(NewDatabaseService),
	fx.Invoke(runMigrations),
	fx.Invoke(registerShutdown),
)

func runMigrations(cfg *configs.Config, db *DatabaseService, logger *slog.Logger) error {
	if !cfg.DB.AutoMigrate {
		logger.Warn("DB_AUTO_MIGRATE disabled; schema must be migrated out-of-band")
		return nil
	}
	return db.GetDB().AutoMigrate(
		&models.User{},
		&models.Account{},
		&models.Session{},
	)
}

func registerShutdown(lc fx.Lifecycle, db *DatabaseService) {
	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return db.Close()
		},
	})
}
