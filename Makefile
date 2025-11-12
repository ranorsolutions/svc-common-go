# ----------------------------------
# Project Metadata
# ----------------------------------
MODULE_NAME := github.com/ranorsolutions/svc-common-go
PKG := ./...
GOFILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# ----------------------------------
# Development Targets
# ----------------------------------

# Format and tidy all Go files
fmt:
	@echo "🧹 Formatting code..."
	@go fmt $(PKG)
	@go mod tidy

# Run all tests with coverage
test:
	@echo "🧪 Running tests..."
	@go test $(PKG) -v -cover -count=1

# Run tests with race detection (slower)
race:
	@echo "🏎️ Running tests with race detector..."
	@go test $(PKG) -race -cover -count=1

# Generate coverage report in HTML
coverage:
	@echo "📊 Generating coverage report..."
	@go test $(PKG) -coverprofile=coverage.out
	@go tool cover -html=coverage.out

# Lint using golangci-lint (if installed)
lint:
	@echo "🔍 Running linters..."
	@golangci-lint run || echo "⚠️ Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

# ----------------------------------
# Dependency Management
# ----------------------------------

# Use local development version of http-common-go
use_dev:
	@echo "🔧 Switching to local http-common-go module..."
	@go mod edit -replace github.com/ranorsolutions/http-common-go=../../../assets/lib/http-common-go

# Use production (remote) version of http-common-go
use_prod:
	@echo "🚀 Switching to production http-common-go module..."
	@go mod edit -dropreplace=github.com/ranorsolutions/http-common-go

# ----------------------------------
# Cleanup
# ----------------------------------

# Remove build/test artifacts
clean:
	@echo "🧼 Cleaning up..."
	@rm -f coverage.out
	@go clean -testcache

# ----------------------------------
# Help
# ----------------------------------

help:
	@echo ""
	@echo "🛠️  Available Make targets:"
	@echo "  fmt          Format code and tidy dependencies"
	@echo "  test         Run tests with coverage"
	@echo "  race         Run tests with race detector"
	@echo "  coverage     Generate coverage HTML report"
	@echo "  lint         Run golangci-lint (if installed)"
	@echo "  use_dev      Replace http-common-go with local version"
	@echo "  use_prod     Drop local replace for http-common-go"
	@echo "  clean        Clean build/test artifacts"
	@echo ""

