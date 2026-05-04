# AGENTS.md - Fiberest Quick Reference

Critical, non-obvious facts for working in this repo. Only includes things an agent would likely miss or get wrong.

---

## Build & Dev Workflow

### Hot reload (task dev) runs `task docs` BEFORE build
`.air.toml` has `pre_cmd = ["task docs"]` — Swagger must be generated before every build in dev mode. If you skip this, the build fails because `docs/docs.go` is imported by `fiber.go`.

### Build order: docs → binary
- `task run` depends on `docs` (via Taskfile)
- `task build` depends on `docs` (via Taskfile)
- `task dev` uses Air with `pre_cmd = ["task docs"]`

Swagger generation command (what Air runs):
```bash
swag init -g cmd/server/main.go -o docs
```

### Go version
`go 1.25.0` — newer than standard. Required by `golang.org/x/crypto` and other deps.

---

## Module Registration (Two Steps)

Creating a module requires BOTH:

1. **Module file** in `internal/modules/{module}/` with `var Module = fx.Options(...)`
2. **Registration** in `internal/modules/combine_module.go`:
   ```go
   import "fiberest/internal/modules/{module}"
   
   var Module = fx.Options(
       {module}.Module,  // Add here
   )
   ```

**Naming note**: `user.module.go` uses `UserRoutes` (not `RegisterRoutes`). Follow existing pattern in each module.

---

## Server Depends on Auth (Critical DI Chain)

`main.go` comment: `server.Module` depends on `auth.AuthService`.

`server/fiber.go`:
- `NewFiberApp(cfg *configs.Config, authService auth.AuthService)` — authService is injected
- Global `app.Use(middlewares.AuthGuard(authService))` — applied to ALL routes
- `auth.Module` has `AutoMigrate` for `Account` and `Session` models

**Consequence**: Auth module MUST be registered and its DB migrations must run before server starts. FX handles this via dependency graph.

---

## Auth Module Auto-Migration

`internal/modules/auth/auth.module.go` includes:
```go
fx.Invoke(func(db *database.DatabaseService) {
    db.GetDB().AutoMigrate(&models.Account{}, &models.Session{})
})
```

This runs automatically when the FX container starts. Don't add similar migrations elsewhere without checking if they belong in the module.

---

## Scalar (Not Swagger UI)

Uses `yokeTH/gofiber-scalar/scalar/v3` — not standard Swagger UI.

Route: `GET /docs/*` returns Scalar interface. Docs generated into `docs/` folder and embedded via `docs.SwaggerInfo.ReadDoc()`.

---

## Port Configuration

- `example.env`: `PORT=3278`
- `.env`: `PORT=3278`
- `server/fiber.go`: reads from `cfg.GetString("PORT")`

**Use**: Set in `.env` (create from `example.env`). Default is 3278.

---

## DTO Import Pattern

Controllers import DTOs from their own module:
```go
package auth

import "fiberest/internal/modules/auth/dto"
```

Not from a shared location. Each module owns its DTOs.

---

## Validation Pattern

Always use the validator:
```go
var req dto.RequestType
if err := validators.ParseAndValidate(ctx, &req); err != nil {
    return validators.ResponseError(ctx, err)
}
```

Located in `internal/common/validators/`. Returns proper HTTP 400 with field errors.

---

## Error Response Helper

Use `http_error` package for consistent errors:
```go
import "fiberest/pkg/http_error"

return http_error.BadRequest(ctx, "message")
return http_error.NotFound(ctx, "message")
return http_error.TooManyRequests(ctx, "message")
```

---

## What to Skip (Generic/Obvious)

- Go naming conventions (PascalCase for exported, camelCase for unexported)
- Basic Fiber route definitions
- Standard GORM usage patterns
- General DI concepts
- Standard error wrapping with `%w`

These are covered by standard Go/Fiber knowledge and don't need repeating here.

---

## Quick Commands

```bash
task dev      # Hot reload (runs docs → build)
task run      # Build docs → run binary
task build    # Build docs → compile binary (./bin/server)
task docs     # Generate Swagger (swag init)
```

---

## References (Keep Updated)

- [Fiber v3 Docs](https://docs.gofiber.io/)
- [Uber FX](https://uber-go.github.io/fx/)
- [GORM](https://gorm.io/docs/)
- [Go Validator](https://github.com/go-playground/validator)
