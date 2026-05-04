package main

import (
	"fiberest/internal/configs"
	"fiberest/internal/database"
	"fiberest/internal/modules"
	"fiberest/internal/server"

	"go.uber.org/fx"
)

// @title Fiberest API
// @description Production-ready Go web framework built on Fiber, designed for enterprise applications with modular architecture, dependency injection, and batteries included.
// @version 1.0.0
// @host localhost:3278
// @BasePath /
// @schemes http

func main() {
	app := fx.New(
		configs.Module,
		database.Module,
		modules.Module, // Provides auth, users, etc.
		server.Module,  // Depends on auth.AuthService
	)
	app.Run()
}
