package logger

import "go.uber.org/fx"

// Module provides *slog.Logger via fx.
var Module = fx.Options(
	fx.Provide(NewLogger),
)
