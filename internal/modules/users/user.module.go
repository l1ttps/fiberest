package users

import (
	"fiberest/internal/common/auth"

	"go.uber.org/fx"
)

// Module exports the user module dependencies
var Module = fx.Options(
	fx.Provide(auth.NewJWTService),
	fx.Provide(NewService),
	fx.Provide(NewController),
	fx.Invoke(UserRoutes),
)
