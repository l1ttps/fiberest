package users

import (
	"go.uber.org/fx"
)

// Module exports the user module dependencies
var Module = fx.Options(
	fx.Provide(UserService),
	fx.Provide(UserController),
	fx.Invoke(UserRoutes),
)
