package modules

import (
	"fiberest/internal/modules/auth"
	"fiberest/internal/modules/users"

	"go.uber.org/fx"
)

var Module = fx.Options(
	auth.Module,
	users.Module,
)
