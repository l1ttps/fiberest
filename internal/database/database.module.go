package database

import (
	"fiberest/internal/models"

	"go.uber.org/fx"
)

// Module provides database-related dependencies
var Module = fx.Options(
	fx.Provide(NewDatabaseService),
	fx.Invoke(func(service *DatabaseService) {}),
	fx.Invoke(func(dbService *DatabaseService) error {
		// Auto-migrate User model to create the users table
		return dbService.GetDB().AutoMigrate(&models.User{})
	}),
)
