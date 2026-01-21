.PHONY: test bench bench-count bench-variance race lint clean build

# Default target
all: test

# Build all packages
build:
	go build ./...

# Run all tests
test:
	go test ./...

# =============================================================================
# Benchmarks - All
# =============================================================================

# Run all benchmarks with memory stats
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

# =============================================================================
# Benchmarks - By Category
# =============================================================================

# Cancel benchmarks (context vs atomic)
bench-cancel:
	go test -bench=BenchmarkCancel -benchmem ./internal/cancel

# Tick benchmarks (ticker implementations)
bench-tick:
	go test -bench=BenchmarkTick -benchmem ./internal/tick

# Queue benchmarks (single goroutine)
bench-queue:
	go test -bench=BenchmarkQueue -benchmem ./internal/queue

# Pipeline benchmarks (2-goroutine SPSC)
bench-pipeline:
	go test -bench=BenchmarkPipeline -benchmem ./internal/combined

# MPSC benchmarks (multiple producers, channel contention)
bench-mpsc:
	go test -bench=BenchmarkMPSC -benchmem ./internal/combined

# go-lock-free-ring comparison benchmarks
bench-lfr:
	go test -bench=BenchmarkLFR -benchmem ./internal/combined

# Combined loop benchmarks (cancel + tick + queue)
bench-combined:
	go test -bench=BenchmarkCombined -benchmem ./internal/combined

# =============================================================================
# Testing & Quality
# =============================================================================

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

# =============================================================================
# Help
# =============================================================================

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Test:"
	@echo "  build          - Build all packages"
	@echo "  test           - Run all tests"
	@echo "  race           - Run tests with race detector"
	@echo "  lint           - Run golangci-lint"
	@echo "  check          - Run build, test, and race"
	@echo ""
	@echo "All Benchmarks:"
	@echo "  bench          - Run all benchmarks with memory stats"
	@echo "  bench-count    - Run benchmarks 10 times (for variance)"
	@echo "  bench-variance - Run benchmarks and save for benchstat"
	@echo "  bench-race     - Run benchmarks with race detector"
	@echo ""
	@echo "Category Benchmarks:"
	@echo "  bench-cancel   - Cancel check: context vs atomic"
	@echo "  bench-tick     - Tick check: ticker implementations"
	@echo "  bench-queue    - Queue: single goroutine push+pop"
	@echo "  bench-pipeline - Pipeline: 2-goroutine SPSC producer/consumer"
	@echo "  bench-mpsc     - MPSC: N producers -> 1 consumer (channel contention)"
	@echo "  bench-lfr      - go-lock-free-ring comparison (SPSC vs MPSC)"
	@echo "  bench-combined - Combined loop: cancel + tick + queue"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean          - Remove generated files"
