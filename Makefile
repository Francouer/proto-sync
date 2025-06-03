.PHONY: build test clean install run help lint

# Build variables
BINARY_NAME=proto-sync
BUILD_DIR=build
MAIN_PATH=./cmd/proto-sync

# Default target
.DEFAULT_GOAL := help

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Binary built at $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## run: Build and run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## run-help: Show application help
run-help: build
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

## run-dry: Run in dry-run mode
run-dry: build
	@./$(BUILD_DIR)/$(BINARY_NAME) --dry-run

## lint: Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

## mod-tidy: Tidy up go.mod
mod-tidy:
	@echo "Tidying go.mod..."
	@go mod tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

## help: Show this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

# Development helpers
.PHONY: dev-setup
## dev-setup: Set up development environment
dev-setup: deps
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ Development environment ready" 