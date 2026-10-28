package auth

import (
	"fiberest/internal/middlewares"

	"go.uber.org/fx"
)

// Module exports the auth module dependencies.
//
// Schema migrations for Account/Session live in database.Module, next to every
// other model, so the whole schema is created in one place.
var Module = fx.Options(
	fx.Provide(NewRepository),
	fx.Provide(NewService),
	// Expose the guard's slice of the service as its own type so other modules
	// can depend on the interface they actually use.
	fx.Provide(func(s AuthService) middlewares.AuthService { return s }),
	fx.Provide(NewController),
	fx.Invoke(RegisterRoutes),
)
