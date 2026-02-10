package main

import (
	"fiberest/internal/configs"
	"fiberest/internal/server"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		configs.Module,
		server.Module,
	)
	app.Run()
}
