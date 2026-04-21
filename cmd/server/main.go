package main

import (
	"fiberest/internal/configs"
	"fiberest/internal/database"
	"fiberest/internal/modules"
	"fiberest/internal/server"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		configs.Module,
		database.Module,
		modules.Module, // Provides auth, users, etc.
		server.Module,  // Depends on auth.AuthService
	)
	app.Run()
}
