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

`docs/` is committed and CI fails on drift (`.github/workflows/ci.yml` job `docs`). After changing any `@Summary`/`@Success` annotation, run `task docs` and commit the result.

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

## Authentication Is Per-Group (No Global Guard)

`app.Use(AuthGuard)` does NOT exist. Each module wraps the routes it protects:

```go
users := app.Group("/users")
users.Patch("/me", middlewares.AuthGuard(authSvc), controller.patchMe)

adminOnly := users.Group("/", middlewares.AuthGuard(authSvc), middlewares.RoleGuard(models.RoleAdmin))
adminOnly.Get("/", controller.getManyUsers)
```

Why: a global guard needs a hand-maintained public-path allow-list and makes unknown paths return 401 instead of 404. Per-group guards remove both problems.

`middlewares.AuthGuard` takes `middlewares.AuthService` (3 methods). `auth.Module` re-exports `auth.AuthService` under that name:

```go
fx.Provide(func(s AuthService) middlewares.AuthService { return s })
```

**Consequence**: a module that registers protected routes must take `middlewares.AuthService` as a parameter of its route function (fx injects it). `internal/server/fx_test.go` validates the whole graph without a database — run it after touching DI.

## Locals accessors

Never read `c.Locals("user_id")` by hand. Use the typed helpers:
```go
user, ok := middlewares.User(ctx)
userID, ok := middlewares.UserID(ctx)
session, ok := middlewares.Session(ctx)
```
They are set by `AuthGuard`; each returns `ok=false` when unauthenticated.

---

## Schema Migrations

One place only: `internal/database/database.module.go` migrates **all** models (`User`, `Account`, `Session`) in a single `AutoMigrate`, gated by `DB_AUTO_MIGRATE`.

- Default: `true` when `APP_ENV != production`, `false` in production.
- Don't add `AutoMigrate` calls inside feature modules.
- Production is expected to migrate out-of-band (versioned migration tool).

`database.Module` also registers `OnStop: db.Close()`; the DB pool is closed on shutdown.

---

## Scalar (Not Swagger UI)

Uses `yokeTH/gofiber-scalar/scalar/v3` — not standard Swagger UI.

Route: `GET /docs/*` returns Scalar interface. Docs generated into `docs/` folder and embedded via `docs.SwaggerInfo.ReadDoc()`.

---

## Configuration (env only, no viper)

`internal/configs/config.service.go` loads config with a small reflection loader over struct tags:

```go
Port int `env:"PORT" default:"3278" validate:"required,min=1,max=65535"`
```

Precedence per key: **process env > `.env` file > `default` tag**. `.env` is loaded with `gotenv.Load`, which never overwrites an existing env var.

- Add a new setting by adding a tagged field — no other wiring.
- Legacy `cfg.GetString("PORT")` accessors are gone; use `cfg.App.Port` etc.
- Conditional defaults resolved in `NewConfig`: `LOG_FORMAT` (json in production), `CORS_ALLOW_ORIGINS` (`*` only in development), `DB_AUTO_MIGRATE`, `SESSION_COOKIE_SECURE`.

Port lives in `cfg.App.Port` (default 3278); set `PORT` in `.env`.

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

```go
// Request body
if err := validators.ParseAndValidate(ctx, &req); err != nil {
    return validators.Invalid(err)
}

// Query parameters (applies types.GetManyRequest.SetDefaults)
if err := validators.GetQuery(ctx, &query); err != nil {
    return validators.Invalid(err)
}

// Path parameters (Fiber's Bind().URI, `param:"id"` tags)
if err := validators.GetParam(ctx, &params); err != nil {
    return validators.Invalid(err)
}
```

- `validators.Invalid(err)` returns a 400 `*http_error.Error` carrying field details.
- Pagination fields must stay optional: `validate:"omitempty,min=1,max=100"` plus `SetDefaults` (the `Defaulter` interface is called by `GetQuery`).
- Path IDs are UUIDs: tag them `validate:"required,uuid"` so a malformed id is a 400, not a Postgres 500.

---

## Errors: Return, Don't Write

There is exactly one place that writes an error body: `http_error.NewHandler`, wired as `fiber.Config.ErrorHandler` in `server.NewFiberApp`.

```go
return http_error.NotFound("user not found")   // 404, message sent to client
return http_error.Internal(err)                // 500, cause logged, client sees "Internal server error"
```

- Never pass `err.Error()` from a DB/service call into a 4xx/5xx message — that leaks internals.
- Controllers map service sentinels with a module-local `mapError(err)` helper.
- `pkg/http_error` must not import `internal/*` (it reads the request id from the `X-Request-ID` header instead).

---

## Pagination

Single shape, single calculation:

```go
types.PageMeta          // Limit, Page, PageCount, HasNextPage, Total — embedded in DTOs
types.NewPageMeta(limit, page, total)
```

Don't compute `pageCount`/`hasNextPage` anywhere else.

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
task test     # go test ./... -race
```

---

## References (Keep Updated)

- [Fiber v3 Docs](https://docs.gofiber.io/)
- [Uber FX](https://uber-go.github.io/fx/)
- [GORM](https://gorm.io/docs/)
- [Go Validator](https://github.com/go-playground/validator)
