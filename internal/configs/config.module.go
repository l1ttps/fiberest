package configs

import "go.uber.org/fx"

// Module provides the configuration dependencies
var Module = fx.Options(
	fx.Provide(NewConfig),
)
