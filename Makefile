.PHONY: help build run test clean plugins deps migrate

# Default target
help:
	@echo "Live Trading API - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make deps              Install dependencies"
	@echo "  make build             Build the API server"
	@echo "  make run               Run the API server"
	@echo "  make test              Run tests"
	@echo "  make plugins           Build all plugin examples"
	@echo "  make clean             Clean build artifacts"
	@echo "  make migrate           Run database migrations manually"
	@echo ""

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Build the API server
build:
	@echo "Building live-trading API server..."
	go build -o bin/live-trading-api cmd/api/main.go
	@echo "Build complete: bin/live-trading-api"

# Run the API server
run:
	@echo "Starting live-trading API server..."
	go run cmd/api/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Build all plugin examples
plugins:
	@echo "Building plugin examples..."
	@mkdir -p plugins
	@cd plugin-examples/grid && go build -buildmode=plugin -o ../../plugins/grid.so .
	@cd plugin-examples/momentum && go build -buildmode=plugin -o ../../plugins/momentum.so .
	@echo "Plugins built in ./plugins/"

# Build the CLI
build-cli:
	@echo "Building kronos-live CLI..."
	go build -o bin/kronos-live cmd/live/main.go
	@echo "CLI build complete: bin/kronos-live"

# Build both server and CLI
build-all: build build-cli

# Run migrations manually (requires DATABASE_CONNECTION_STRING env var)
migrate:
	@echo "Running database migrations..."
	@go run cmd/api/main.go migrate

# Development mode with hot reload (requires air)
dev:
	@command -v air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	@air

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@command -v golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...

# Build for production (optimized)
build-prod:
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/live-trading-api cmd/api/main.go
	@echo "Production build complete: bin/live-trading-api"

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t live-trading-api:latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8081:8081 --env-file .env live-trading-api:latest
