package modules

import (
	"fiberest/internal/modules/auth"
	healthcheck "fiberest/internal/modules/health_check"
	"fiberest/internal/modules/users"

	"go.uber.org/fx"
)

var Module = fx.Options(
	healthcheck.Module,
	auth.Module,
	users.Module,
)
