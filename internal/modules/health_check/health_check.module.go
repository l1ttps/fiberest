package healthcheck

import (
	"go.uber.org/fx"
)

// Module exports the health check module dependencies
var Module = fx.Options(
	fx.Provide(NewService),
	fx.Provide(NewController),
	fx.Invoke(RegisterRoutes),
)
