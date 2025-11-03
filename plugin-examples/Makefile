.PHONY: all clean grid momentum test

# Build all plugins
all: grid momentum

# Build grid plugin
grid:
	@echo "Building grid.so plugin..."
	cd grid && go build -buildmode=plugin -o ../grid.so .
	@echo "✓ grid.so built successfully"

# Build momentum plugin
momentum:
	@echo "Building momentum.so plugin..."
	cd momentum && go build -buildmode=plugin -o ../momentum.so .
	@echo "✓ momentum.so built successfully"

# Clean built plugins
clean:
	@echo "Cleaning plugins..."
	rm -f grid.so momentum.so
	@echo "✓ Plugins cleaned"

# Test build (check compilation without creating plugins)
test:
	@echo "Testing grid strategy..."
	cd grid && go build -buildmode=plugin -o /tmp/grid_test.so .
	@echo "✓ Grid strategy compiles"
	@echo "Testing momentum strategy..."
	cd momentum && go build -buildmode=plugin -o /tmp/momentum_test.so .
	@echo "✓ Momentum strategy compiles"
	@rm -f /tmp/grid_test.so /tmp/momentum_test.so

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "✓ Dependencies downloaded"

# Verify Go version
check-version:
	@echo "Checking Go version..."
	@go version | grep -q "go1.24" || (echo "ERROR: Go 1.24.x required for plugin compatibility" && exit 1)
	@echo "✓ Go version compatible"

# Full build with checks
build: check-version deps all
	@echo "✓ All plugins built and ready"
