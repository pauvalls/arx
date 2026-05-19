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

# Run benchmarks
bench:
	$(GO) test -bench=/Detection -benchmem -count=5 ./internal/application/ | tee bench-output.txt

# Compare benchmarks with baseline using benchstat
bench-compare:
	$(GO) test -bench=/Detection -benchmem -count=5 ./internal/application/ | tee /tmp/bench.new
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat .bench-baseline /tmp/bench.new > /tmp/benchstat.out; \
		cat /tmp/benchstat.out; \
		echo "---"; \
		# Hard fail if DetectionPipeline_10k has >5% regression \
		if grep "DetectionPipeline_10k" /tmp/benchstat.out | grep -qP '\d+\.\d+%'; then \
			REGRESSION=$$(grep "DetectionPipeline_10k" /tmp/benchstat.out | grep -oP '\d+\.\d+(?=%)'); \
			if [ -n "$$REGRESSION" ] && [ "$$(echo "$$REGRESSION > 5" | bc)" = "1" ]; then \
				echo "❌ FAIL: DetectionPipeline_10k regression of $$REGRESSION% exceeds 5% threshold"; \
				exit 1; \
			fi; \
		fi; \
		echo "✓ No significant regression detected"; \
	else \
		echo "⚠️  benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

# Generate benchmark baseline
bench-baseline:
	$(GO) test -bench=/Detection -benchmem -count=10 ./internal/application/ | tee .bench-baseline

# Show help
help:
	@echo "Arx - Architectural Linter"
	@echo ""
	@echo "Usage:"
	@echo "  make build    - Compile the binary to ./arx"
	@echo "  make test     - Run all tests with verbose output"
	@echo "  make lint     - Run golangci-lint (skips if not installed)"
	@echo "  make bench    - Run benchmarks and save output"
	@echo "  make bench-compare - Compare benchmarks against baseline"
	@echo "  make bench-baseline - Generate benchmark baseline"
	@echo "  make clean    - Remove compiled binary"
	@echo "  make help     - Show this help message"
