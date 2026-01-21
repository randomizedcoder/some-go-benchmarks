.PHONY: test bench bench-count bench-variance race lint clean build

# Default target
all: test

# Build all packages
build:
	go build ./...

# Run all tests
test:
	go test ./...

# Run benchmarks with memory stats
bench:
	go test -bench=. -benchmem ./internal/...

# Run benchmarks with multiple iterations (for variance analysis)
bench-count:
	go test -bench=. -benchmem -count=10 ./internal/...

# Run specific benchmark with variance check
bench-variance:
	@echo "Running benchmarks 10 times for variance analysis..."
	go test -bench=. -count=10 ./internal/... | tee bench_results.txt
	@echo ""
	@echo "Analyze with: benchstat bench_results.txt"

# Run tests with race detector
race:
	go test -race ./...

# Run linter
lint:
	golangci-lint run ./...

# Run benchmarks with race detector (slower)
bench-race:
	go test -race -bench=. -benchtime=100ms ./internal/...

# Clean build artifacts
clean:
	rm -f bench_results.txt
	rm -f *.prof
	rm -f *.test

# Quick sanity check
check: build test race
	@echo "All checks passed!"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build all packages"
	@echo "  test          - Run all tests"
	@echo "  bench         - Run benchmarks with memory stats"
	@echo "  bench-count   - Run benchmarks 10 times"
	@echo "  bench-variance- Run benchmarks and save for benchstat"
	@echo "  race          - Run tests with race detector"
	@echo "  lint          - Run golangci-lint"
	@echo "  bench-race    - Run benchmarks with race detector"
	@echo "  clean         - Remove generated files"
	@echo "  check         - Run build, test, and race"
