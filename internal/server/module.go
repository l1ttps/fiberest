package server

import "go.uber.org/fx"

// Module provides the server dependencies and lifecycle hooks
var Module = fx.Options(
	fx.Provide(NewFiberApp),
	fx.Invoke(RegisterFiberLifecycle),
)
