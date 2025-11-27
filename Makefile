.PHONY: help build deps

# Default target
help:
	@echo "Live Trading API - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make deps              Install dependencies"
	@echo "  make build             Build the API server"
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
