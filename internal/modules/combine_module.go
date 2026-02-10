package modules

import (
	healthcheck "fiberest/internal/modules/health_check"

	"go.uber.org/fx"
)

var Module = fx.Options(
	healthcheck.Module,
)
