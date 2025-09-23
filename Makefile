# Crescendo Makefile
# Provides convenient commands for testing, building, and development

.PHONY: test test-verbose test-coverage test-race test-bench clean build run server dev

# Default target
test:
	@echo "🧪 Running tests..."
	@go test ./...

# Verbose test output
test-verbose:
	@echo "🧪 Running tests with verbose output..."
	@go test -v ./...

# Test with coverage reporting
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep -E "(total:|main\.go)"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📈 Coverage report generated: coverage.html"

# Test for race conditions
test-race:
	@echo "🏁 Running race condition tests..."
	@go test -race ./...

# Run benchmarks
test-bench:
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Run comprehensive test suite
test-all: test-coverage test-race test-bench
	@echo "✅ Comprehensive test suite complete!"

# Clean test artifacts
clean:
	@echo "🧹 Cleaning up test artifacts..."
	@rm -f coverage.out coverage.html

# Build the application
build:
	@echo "🔨 Building Crescendo..."
	@go build -o crescendo .

# Run CLI mode
run:
	@echo "🚀 Running Crescendo CLI..."
	@go run .

# Run server mode
server:
	@echo "🌐 Starting Crescendo server..."
	@go run . --server --port 8080

# Development mode with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "🔥 Starting development server with hot reload..."
	@air || echo "⚠️  Air not installed. Run: go install github.com/cosmtrek/air@latest"

# Run automated test script
test-script:
	@./test.sh

# Open coverage report in browser (macOS)
test-open:
	@./test.sh --open

# Help
help:
	@echo "Crescendo Makefile Commands:"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run basic tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-coverage- Run tests with coverage reporting"
	@echo "  make test-race    - Run race condition tests"
	@echo "  make test-bench   - Run performance benchmarks"
	@echo "  make test-all     - Run comprehensive test suite"
	@echo "  make test-script  - Run automated test script"
	@echo "  make test-open    - Run tests and open coverage report"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run CLI mode"
	@echo "  make server       - Run server mode"
	@echo "  make dev          - Start development server with hot reload"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean        - Clean test artifacts"
	@echo "  make help         - Show this help"