# Fiberest Boilerplate

This is a boilerplate for projects using Go, Fiber, and GORM, built with a modular architecture and dependency injection.

## Directory Structure

The project is organized as follows:

```
.
├── cmd
│   ├── server      # Contains the main function to start the server
│   └── swag        # Swagger configuration
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
├── Makefile        # Contains commands to build, run, and manage the project
└── ...
```

## Setup Guide

### Prerequisites

- [Go](https://golang.org/dl/) (version 1.22.0 or newer)
- [PostgreSQL](https://www.postgresql.org/download/)
- [Air](https://github.com/cosmtrek/air) (optional, for live-reloading)

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
    - **Development mode:**
      This command starts the server.

      ```bash
      task dev
      ```

    - **Normal execution:**
      This command will also start the application.

      ```bash
      task run
      ```

## Useful Commands

- `task build`: Builds the application into an executable file in the `bin` directory.
- `task swagger`: Generates Swagger documentation.

## API Documentation

The project uses Swagger for API documentation. After starting the server, you can access the `/swagger` path to see the details.

Example: `http://localhost:3278/swagger`
