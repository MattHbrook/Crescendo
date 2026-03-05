.PHONY: all build test test-v test-integration lint sec fmt tidy clean docker help

BINARY := build/crescendo
GO := go
GOFLAGS := -race -count=1
LDFLAGS := -ldflags="-s -w"

## all: run all quality gates (fmt + lint + test + sec + build)
all: fmt lint test sec build

## build: compile to build/crescendo
build:
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/crescendo

## test: run tests with race detector
test:
	$(GO) test $(GOFLAGS) ./...

## test-v: run tests with verbose output
test-v:
	$(GO) test $(GOFLAGS) -v ./...

## test-integration: run integration tests
test-integration:
	$(GO) test $(GOFLAGS) -tags=integration ./...

## lint: run golangci-lint
lint:
	golangci-lint run

## sec: run govulncheck
sec:
	govulncheck ./...

## fmt: format code with gofmt and goimports
fmt:
	gofmt -w .
	goimports -w .

## tidy: tidy and verify go.mod
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## clean: remove build artifacts
clean:
	rm -rf build/

## docker: build Docker image locally
docker:
	docker build -t crescendo:local .

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
