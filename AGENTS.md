# AGENTS.md - Fiberest Project Guidelines

> **Project**: Fiberest - Go Fiber Boilerplate with Dependency Injection
> **Language**: Go (Golang)
> **Framework**: Fiber v3, Uber FX (DI), GORM, Viper

---

## 1. Architecture Overview

This project follows **Modular Architecture** combined with **Dependency Injection** using Uber FX.

```
┌─────────────────────────────────────────────────────────┐
│                        main.go                          │
│              (Initialize FX container)                  │
└────────────────────┬────────────────────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    │                │                │
┌───▼────┐    ┌──────▼──────┐   ┌─────▼──────┐
│ Config │    │  Database   │   │   Server   │
│ Module │    │   Module    │   │   Module   │
└───┬────┘    └──────┬──────┘   └─────┬──────┘
    │                │                │
    └────────────────┴────────────────┘
                     │
            ┌────────▼─────────┐
            │  Modules Module  │
            │  (Combine all)   │
            └────────┬─────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
   ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
   │ Module  │  │ Module  │  │  ...    │
   │    A    │  │    B    │  │ Modules │
   └─────────┘  └─────────┘  └─────────┘
```

---

## 2. Project Structure

```
.
├── cmd/
│   ├── server/
│   │   └── main.go              # Entry point - initialize FX container
│   └── swag/
│       └── docs/                # Swagger generated docs
├── internal/
│   ├── common/                  # Shared components
│   │   ├── models/              # Base models (BaseModel with UUID, timestamps)
│   │   ├── types/               # Shared types, constants
│   │   └── validators/          # Request validation utilities
│   ├── configs/                 # Configuration management
│   │   ├── config.module.go     # FX module
│   │   └── config.service.go    # Config service using Viper
│   ├── database/                # Database connection
│   │   ├── database.module.go   # FX module
│   │   └── database.service.go  # GORM service
│   ├── modules/                 # Business modules
│   │   ├── combine_module.go    # Aggregate all modules
│   │   └── {module}/            # Each business module
│   │       ├── dto/             # Data Transfer Objects
│   │       ├── models/          # GORM models
│   │       ├── {module}.module.go
│   │       ├── {module}.controller.go
│   │       └── {module}.service.go
│   └── server/                  # HTTP server setup
│       ├── fiber.go             # Fiber app & lifecycle
│       └── module.go            # Server module
├── pkg/                         # Public packages
│   └── http_error/              # HTTP error handling
└── docs/                        # Documentation
```

---

## 3. Module Pattern

Each module follows this pattern:

### 3.1 File Structure

```
internal/modules/{module_name}/
├── {module}.module.go      # Module definition
├── {module}.controller.go  # HTTP handlers
├── {module}.service.go     # Business logic
├── dto/                    # Request/Response structs
└── models/                 # Database models (if any)
```

### 3.2 Module Definition (`{module}.module.go`)

```go
package {module_name}

import "go.uber.org/fx"

var Module = fx.Options(
    fx.Provide(NewService),      // Provide service
    fx.Provide(NewController),   // Provide controller
    fx.Invoke(RegisterRoutes),   // Register routes
)
```

### 3.3 Controller Pattern

```go
package {module_name}

import "github.com/gofiber/fiber/v3"

type Controller struct {
    service *Service  // Inject service via DI
}

func NewController(app *fiber.App, service *Service) *Controller {
    return &Controller{service: service}
}

func RegisterRoutes(app *fiber.App, controller *Controller) {
    group := app.Group("/endpoint")
    group.Post("/action", controller.handleAction)
}

func (c *Controller) handleAction(ctx fiber.Ctx) error {
    // 1. Parse & validate request
    // 2. Call service
    // 3. Return response
}
```

### 3.4 Service Pattern

```go
package {module_name}

type Service struct {
    dbService *database.DatabaseService  // Inject dependencies
}

func NewService(dbService *database.DatabaseService) *Service {
    return &Service{dbService: dbService}
}

// Public methods use PascalCase
func (s *Service) PublicMethod() error {
    // Business logic
}

// Private helpers use camelCase
func (s *Service) privateHelper() error {
    // Helper logic
}
```

---

## 4. Coding Standards

### 4.1 Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Package | lowercase, no underscore | `users`, `orders` |
| Exported | PascalCase | `UserService`, `CreateUser` |
| Unexported | camelCase | `hashPassword`, `getDB` |
| Constants | UPPER_SNAKE_CASE | `RoleAdmin`, `DefaultPort` |
| Files | snake_case | `user.service.go` |
| DTOs | Suffix with Request/Response | `CreateUserRequest` |

### 4.2 Error Handling

```go
// Always wrap errors with context
if err != nil {
    return nil, fmt.Errorf("failed to create user: %w", err)
}

// Check specific errors
if err == gorm.ErrRecordNotFound {
    return nil, fmt.Errorf("user not found")
}

// Return validation errors
return http_error.BadRequest(ctx, "email already exists")
```

### 4.3 Request Validation

```go
var req dto.RequestType
if err := validators.ParseAndValidate(ctx, &req); err != nil {
    return validators.ResponseError(ctx, err)
}
```

### 4.4 Database Operations

```go
// Use transaction when multiple operations
err := s.getDB().Transaction(func(tx *gorm.DB) error {
    // operations...
    return nil
})

// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

---

## 5. Creating a New Module

### Step 1: Create directory structure

```bash
mkdir -p internal/modules/{module_name}/{dto,models}
touch internal/modules/{module_name}/{module_name}.module.go
touch internal/modules/{module_name}/{module_name}.controller.go
touch internal/modules/{module_name}/{module_name}.service.go
```

### Step 2: Implement module.go

```go
package {module_name}

import "go.uber.org/fx"

var Module = fx.Options(
    fx.Provide(NewService),
    fx.Provide(NewController),
    fx.Invoke(RegisterRoutes),
)
```

### Step 3: Register in combine_module.go

```go
import "fiberest/internal/modules/{module_name}"

var Module = fx.Options(
    // existing modules...
    {module_name}.Module,
)
```

---

## 6. Swagger Documentation

All controller handlers must have Swagger annotations:

```go
// @Summary Short description
// @Description Detailed description
// @Tags {module_name}
// @Accept json
// @Produce json
// @Param request body dto.Request true "Request description"
// @Success 200 {object} dto.Response
// @Failure 400 {object} http_error.ErrorResponse
// @Router /path [method]
func (c *Controller) handler(ctx fiber.Ctx) error {
    // implementation
}
```

### Regenerate Swagger

```bash
make swag
# or
task swag
```

---

## 7. Dependencies & Tools

### Core Dependencies

- `github.com/gofiber/fiber/v3` - Web framework
- `go.uber.org/fx` - Dependency injection
- `gorm.io/gorm` - ORM
- `gorm.io/driver/postgres` - PostgreSQL driver
- `github.com/spf13/viper` - Configuration management
- `github.com/go-playground/validator/v10` - Validation

### Available Commands

```bash
# Development (hot reload)
make dev

# Build
make build

# Run tests
make test

# Generate Swagger docs
make swag

# Clean build artifacts
make clean
```

---

## 8. Environment Variables

```bash
# Server
PORT=3278

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=secret
DB_NAME=fiberest
DB_SSLMODE=disable
```

---

## 9. Key Principles

1. **Single Responsibility**: Each module has one responsibility only
2. **Dependency Injection**: Always use FX for dependency injection
3. **Explicit Error Handling**: Never ignore errors, always wrap with context
4. **Validation First**: Validate requests before business logic
5. **No Business Logic in Controller**: Controller handles HTTP only
6. **No HTTP in Service**: Service handles business logic only
7. **DTO Pattern**: Always use DTOs for request/response
8. **Security**: Hash passwords, don't expose sensitive data in JSON

---

## 10. References

- [Fiber Documentation](https://docs.gofiber.io/)
- [Uber FX Documentation](https://uber-go.github.io/fx/)
- [GORM Documentation](https://gorm.io/docs/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
