# Fiberest

Production-ready Go web framework built on Fiber, designed for enterprise applications with modular architecture, dependency injection, and batteries included.

## Table of Contents
- [Directory Structure](#directory-structure)
- [Setup Guide](#setup-guide)
  - [Prerequisites](#prerequisites)
  - [Installation Steps](#installation-steps)
- [Usage](#usage)
  - [Creating a New Module](#creating-a-new-module)
  - [Adding Routes](#adding-routes)
  - [Dependency Injection](#dependency-injection)
  - [Working with Database](#working-with-database)
  - [Request Validation](#request-validation)
  - [Middlewares](#middlewares)
  - [Generating API Documentation](#generating-api-documentation)
- [Useful Commands](#useful-commands)
- [API Documentation](#api-documentation)

## Directory Structure

The project is organized as follows:

```
.
├── cmd
│   └── server      # Contains the main function to start the server
├── docs            # Generated API documentation
├── internal
│   ├── common      # Shared components (models, types, validators)
│   ├── configs     # Configuration management module (using Viper)
│   ├── database    # Database connection management module (using GORM)
│   ├── modules     # Contains business logic modules (e.g., users, health_check)
│   └── server      # Fiber server configuration and initialization
├── pkg
│   └── http_error  # Helper for handling HTTP errors
├── .air.toml       # Configuration for live-reloading with Air
├── .env            # File for environment variables (should be created from example.env)
├── example.env     # Example file for environment variables
├── go.mod          # Declares the Go module and its dependencies
├── taskfile.yml    # Contains commands to build, run, and manage the project
└── ...
```

## Setup Guide

### Prerequisites

- [Go](https://golang.org/dl/) (version 1.25.0 or newer)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Air](https://github.com/cosmtrek/air) (optional, for live-reloading)
- [Task](https://taskfile.dev/) (required for running commands)
- [Swag](https://github.com/swaggo/swag) (required for generating docs)

### Installation Steps

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/l1ttps/fiberest
    cd fiberest
    ```

2.  **Install dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Configure environment variables:**

    Create a `.env` file from the `example.env` file and update the values to match your environment.

    ```bash
    cp example.env .env
    ```

    The `.env` file will look like this:

    ```env
    PORT=3278

    # PostgreSQL Configuration
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=postgres
    DB_PASSWORD=your_password
    DB_NAME=fiberest
    DB_SSLMODE=disable
    ```

4.  **Run the project:**

    Use `task` to run the project.
    - **Development mode (with hot reload):**

      ```bash
      task dev
      ```

    - **Normal execution:**

      ```bash
      task run
      ```

## Usage

### Creating a New Module

This project follows a modular architecture. Each business feature should be organized as a separate module in `internal/modules/`.

To create a new module:

1.  **Create directory structure:**
    ```bash
    mkdir -p internal/modules/{module_name}/{dto,models}
    ```

2.  **Create core files:**
    ```bash
    touch internal/modules/{module_name}/{module_name}.module.go
    touch internal/modules/{module_name}/{module_name}.controller.go
    touch internal/modules/{module_name}/{module_name}.service.go
    ```

3.  **Module definition (`{module}.module.go`):**
    ```go
    package {module_name}

    import "go.uber.org/fx"

    var Module = fx.Options(
        fx.Provide(NewService),      // Provide business logic service
        fx.Provide(NewController),   // Provide HTTP controller
        fx.Invoke(RegisterRoutes),   // Register HTTP routes
    )
    ```

4.  **Register module:**
    Add your module to `internal/modules/combine_module.go`:
    ```go
    import "fiberest/internal/modules/{module_name}"

    var Module = fx.Options(
        // ... existing modules
        {module_name}.Module,
    )
    ```

### Adding Routes

Routes are registered in the controller's `RegisterRoutes` function:

```go
func RegisterRoutes(app *fiber.App, controller *Controller) {
    group := app.Group("/api/v1/{endpoint}")
    
    group.Post("", controller.Create)    // POST /api/v1/{endpoint}
    group.Get("", controller.List)       // GET /api/v1/{endpoint}
    group.Get("/:id", controller.Get)    // GET /api/v1/{endpoint}/:id
    group.Put("/:id", controller.Update) // PUT /api/v1/{endpoint}/:id
    group.Delete("/:id", controller.Delete) // DELETE /api/v1/{endpoint}/:id
}
```

### Dependency Injection

The project uses Uber FX for dependency injection. All dependencies are automatically resolved:

```go
// Service depends on DatabaseService
func NewService(dbService *database.DatabaseService) *Service {
    return &Service{dbService: dbService}
}

// Controller depends on Service
func NewController(service *Service) *Controller {
    return &Controller{service: service}
}
```

FX automatically handles the lifecycle and injection of all provided components.

### Working with Database

Use the injected `DatabaseService` to access GORM:

```go
// Get database instance
db := s.dbService.DB

// Create record
err := db.Create(&user).Error

// Find record
err := db.First(&user, id).Error

// Update record
err := db.Model(&user).Updates(User{Name: "New Name"}).Error

// Delete record
err := db.Delete(&user, id).Error

// Transactions
err := db.Transaction(func(tx *gorm.DB) error {
    // Multiple operations that must succeed together
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    return tx.Create(&profile).Error
})
```

### Request Validation

Use the built-in validator to validate incoming requests:

```go
import "fiberest/internal/common/validators"

func (c *Controller) Create(ctx fiber.Ctx) error {
    var req dto.CreateUserRequest
    if err := validators.ParseAndValidate(ctx, &req); err != nil {
        return validators.ResponseError(ctx, err)
    }
    
    // Process valid request
}
```

### Middlewares

Fiberest provides pre-built middlewares for common security and rate limiting needs.

#### AuthGuard Middleware

JWT-based authentication middleware that protects routes. Automatically checks for tokens in cookies (`access_token`) or `Authorization: Bearer` header.

```go
import "fiberest/internal/middlewares"

// Apply globally (recommended) - protects all routes except public ones
app.Use(middlewares.AuthGuard(config))

// Apply to specific route groups
group := app.Group("/api/v1/protected", middlewares.AuthGuard(config))
```

Public routes (bypass authentication) are defined in `internal/middlewares/auth_guard.go`. Supports exact matches, prefix wildcards (`/public/*`), and suffix wildcards (`*.html`).

#### Rate Limiter Middleware

Rate limiting middleware to prevent abuse and protect API endpoints.

```go
import "fiberest/internal/middlewares"

// Allow 100 requests per minute
app.Use(middlewares.Limiter(100, 60))

// Apply stricter limits to sensitive endpoints
group.Post("/login", middlewares.Limiter(5, 60), controller.Login)
```

### Generating API Documentation

Add Swagger annotations to your controller handlers:

```go
// @Summary Create a new user
// @Description Creates a new user in the system
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "User data"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} http_error.ErrorResponse
// @Failure 409 {object} http_error.ErrorResponse
// @Router /api/v1/users [post]
func (c *Controller) Create(ctx fiber.Ctx) error {
    // implementation
}
```

Generate updated documentation:
```bash
task docs
```

## Useful Commands

- `task build`: Builds the application into an executable file in the `bin` directory.
- `task docs`: Generates API documentation in `/docs` folder.

## API Documentation

The project uses Scalar for API documentation. After starting the server, you can access the `/docs` path to see the details.

Example: `http://localhost:3278/docs`
