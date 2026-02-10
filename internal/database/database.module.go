package database

import (
	"go.uber.org/fx"
)

// Module provides database-related dependencies
var Module = fx.Options(
	fx.Provide(NewDatabaseService),
	fx.Invoke(func(service *DatabaseService) {}),
)
