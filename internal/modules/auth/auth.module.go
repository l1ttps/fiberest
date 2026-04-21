package auth

import (
	"fiberest/internal/database"
	"fiberest/internal/models"

	"go.uber.org/fx"
)

// Module exports the auth module dependencies
var Module = fx.Options(
	fx.Provide(NewService),
	fx.Provide(NewController),
	fx.Invoke(RegisterRoutes),
	// Database auto-migration for auth models
	fx.Invoke(func(db *database.DatabaseService) {
		db.GetDB().AutoMigrate(
			&models.Account{},
			&models.Session{},
		)
	}),
)
