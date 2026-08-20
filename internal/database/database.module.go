package database

import (
	"fiberest/internal/configs"
	"fiberest/internal/models"
	"log/slog"

	"go.uber.org/fx"
)

type dbParams struct {
	fx.In
	Config *configs.Config
	Logger *slog.Logger `optional:"true"`
}

func newDatabaseServiceFromFX(p dbParams) (*DatabaseService, error) {
	return NewDatabaseService(p.Config, p.Logger)
}

// Module provides database-related dependencies
var Module = fx.Options(
	fx.Provide(newDatabaseServiceFromFX),
	fx.Invoke(func(service *DatabaseService) {}),
	fx.Invoke(func(dbService *DatabaseService) error {
		return dbService.GetDB().AutoMigrate(&models.User{})
	}),
)
