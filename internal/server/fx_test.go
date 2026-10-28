package server_test

import (
	"testing"

	"fiberest/internal/configs"
	"fiberest/internal/database"
	"fiberest/internal/logger"
	"fiberest/internal/modules"
	"fiberest/internal/server"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// TestFxGraph validates the whole dependency graph (and route registration)
// without touching a real database. It catches wiring regressions — e.g. a
// module asking for an interface nobody provides — that would otherwise only
// surface when the binary starts.
func TestFxGraph(t *testing.T) {
	t.Setenv("DB_AUTO_MIGRATE", "false")

	err := fx.ValidateApp(
		// No PostgreSQL in unit tests: swap the constructor, keep the type.
		fx.Replace(fx.Annotate(func() *database.DatabaseService {
			return &database.DatabaseService{}
		})),
		configs.Module,
		logger.Module,
		database.Module,
		modules.Module,
		server.Module,
	)
	require.NoError(t, err)
}
