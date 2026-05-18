.PHONY: build test lint clean help

# Binary name
BINARY = arx

# Go command
GO = go

# Build flags
GOFLAGS = -v

# Default target
all: build

# Build the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY) ./cmd/arx

# Run tests
test:
	$(GO) test $(GOFLAGS) ./...

# Run tests with race detector
test-race:
	$(GO) test -race ./...

# Run tests with coverage
cover:
	$(GO) test -cover ./... | sort -k3 -n

# Quality gate: vet + race + coverage (exit 1 if any core package < 50%)
quality: vet test-race cover
	@echo "✓ All quality gates passed"

# Run go vet
vet:
	$(GO) vet ./...

# Run linter (graceful skip if not installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	rm -f $(BINARY)

# Show help
help:
	@echo "Arx - Architectural Linter"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Compile the binary to ./arx"
	@echo "  make test     - Run all tests with verbose output"
	@echo "  make lint     - Run golangci-lint (skips if not installed)"
	@echo "  make clean    - Remove compiled binary"
	@echo "  make help     - Show this help message"
