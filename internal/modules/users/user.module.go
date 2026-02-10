package users

import (
	"go.uber.org/fx"
)

// Module exports the user module dependencies
var Module = fx.Options(
	fx.Provide(NewService),
	fx.Provide(NewController),
	fx.Invoke(RegisterUserRoutes),
)
