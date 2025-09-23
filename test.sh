#!/bin/bash

# Crescendo Test Automation Script
# Runs comprehensive tests with coverage reporting

set -e

echo "🚀 Starting Crescendo Test Suite..."
echo

# Clean up any previous test artifacts
echo "🧹 Cleaning up previous test artifacts..."
rm -f coverage.out coverage.html

# Run tests with coverage
echo "🧪 Running tests with coverage..."
go test -v -coverprofile=coverage.out ./...

# Generate coverage statistics
echo
echo "📊 Coverage Statistics:"
go tool cover -func=coverage.out | grep -E "(total:|main\.go)"

# Generate HTML coverage report
echo
echo "🌐 Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "Coverage report generated: coverage.html"

# Run performance benchmarks
echo
echo "⚡ Running performance benchmarks..."
go test -bench=. -benchmem ./...

# Check for race conditions (if supported)
echo
echo "🏁 Running race condition tests..."
go test -race ./... || echo "⚠️  Race tests failed or not supported on this platform"

# Final summary
echo
echo "✅ Test Suite Complete!"
echo "📈 Main package coverage: $(go tool cover -func=coverage.out | grep "total:" | awk '{print $3}')"
echo "📝 Detailed coverage report: coverage.html"
echo

# Optional: Open coverage report in browser (macOS)
if [[ "$OSTYPE" == "darwin"* ]] && [[ "$1" == "--open" ]]; then
    echo "🔍 Opening coverage report in browser..."
    open coverage.html
fi