package modules

import (
	healthcheck "fiberest/internal/modules/health_check"
	"fiberest/internal/modules/users"

	"go.uber.org/fx"
)

var Module = fx.Options(
	healthcheck.Module,
	users.Module,
)
