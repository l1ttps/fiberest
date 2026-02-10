package main

import (
	"fiberest/internal/configs"
	"fiberest/internal/modules"
	"fiberest/internal/server"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		configs.Module,
		server.Module,
		modules.Module,
	)
	app.Run()
}
