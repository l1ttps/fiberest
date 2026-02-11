# Go parameters
GOCMD = go
GOBIN = ./bin
TARGET = ./cmd/server/main

# Build output
BINARY_NAME = server
BINARY_DIR = ./bin

.PHONY: all dev build clean run

# Default target
all: help

dev:
	@echo "Starting development server with air..."
	air

build:
	@echo "Building application..."
	$(GOCMD) build -o $(BINARY_DIR)/$(BINARY_NAME) $(TARGET)
	@echo "Build completed: $(BINARY_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	@echo "Clean completed"

run: build
	@echo "Running application..."
	$(BINARY_DIR)/$(BINARY_NAME)

help:
	@echo "Available targets:"
	@echo "  dev    - Start development server with live reload (air)"
	@echo "  build  - Build the application"
	@echo "  run    - Build and run the application"
	@echo "  clean  - Clean build artifacts"
