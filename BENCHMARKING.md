# Benchmarking Guide

This document provides guidance for running and interpreting benchmarks.

## Quick Start

```bash
# Run all benchmarks
make bench

# Run with multiple iterations for variance analysis
make bench-count

# Run specific package
go test -bench=. -benchmem ./internal/cancel
```

## Environment Setup

### Linux (Recommended)

For consistent, reproducible results:

```bash
# 1. Set CPU governor to performance (prevents frequency scaling)
sudo cpupower frequency-set -g performance

# 2. Disable turbo boost (for consistent clock speed)
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# 3. Verify CPU frequency is stable
watch -n1 "cat /proc/cpuinfo | grep MHz | head -4"

# 4. Check for background processes
top -bn1 | head -20
```

### GOMAXPROCS

Control how many OS threads execute Go code:

```bash
# Single-threaded execution (lowest variance, no goroutine scheduling noise)
GOMAXPROCS=1 go test -bench=. ./internal/...

# Match physical cores (no hyperthreading)
GOMAXPROCS=4 go test -bench=. ./internal/...

# Default: uses all logical CPUs (GOMAXPROCS=runtime.NumCPU())
go test -bench=. ./internal/...
```

**When to use:**
- `GOMAXPROCS=1`: Best for measuring raw single-threaded performance
- `GOMAXPROCS=N`: For parallel benchmarks (`b.RunParallel`)
- Default: For realistic multi-core scenarios

### Pinning to Single Core (Lowest Variance)

```bash
# Run on CPU 0 only
taskset -c 0 go test -bench=. ./internal/...

# Combined: single core + single GOMAXPROCS (ultimate isolation)
taskset -c 0 GOMAXPROCS=1 go test -bench=. ./internal/...
```

### Scheduler Priority (nice/renice)

Increase process priority to reduce interference from other processes:

```bash
# Run with highest priority (requires root)
sudo nice -n -20 go test -bench=. ./internal/...

# Or renice an existing process
sudo renice -n -20 -p $(pgrep -f "go test")
```

**Nice values:**
- `-20`: Highest priority (most CPU time)
- `0`: Default priority
- `19`: Lowest priority (least CPU time)

**Combined with CPU pinning for maximum isolation:**

```bash
sudo nice -n -20 taskset -c 0 GOMAXPROCS=1 go test -bench=. ./internal/...
```

> **Note:** High priority alone doesn't prevent context switches. For true isolation, combine with CPU pinning and consider isolating CPU cores from the scheduler (`isolcpus` kernel parameter).

### macOS

```bash
# Disable App Nap (can affect timing)
defaults write NSGlobalDomain NSAppSleepDisabled -bool YES

# Run with elevated priority (macOS equivalent of nice)
sudo nice -n -20 go test -bench=. ./internal/...
```

### Advanced: Kernel-Level CPU Isolation

For the most stable benchmarks on dedicated machines:

```bash
# 1. Add to kernel boot parameters (GRUB)
#    isolcpus=2,3 nohz_full=2,3 rcu_nocbs=2,3

# 2. After reboot, CPUs 2-3 are isolated from scheduler
#    Run benchmarks on isolated CPU:
sudo taskset -c 2 nice -n -20 GOMAXPROCS=1 go test -bench=. ./internal/...
```

This removes the CPUs from general scheduling entirely.

## Running Benchmarks

### Standard Run

```bash
go test -bench=. -benchmem ./internal/...
```

### With Variance Analysis

Run 10 iterations and analyze with `benchstat`:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks
go test -bench=. -count=10 ./internal/... > results.txt

# Analyze
benchstat results.txt
```

### Comparing Before/After

```bash
# Before changes
go test -bench=. -count=10 ./internal/... > old.txt

# Make changes...

# After changes
go test -bench=. -count=10 ./internal/... > new.txt

# Compare
benchstat old.txt new.txt
```

## Interpreting Results

### Understanding Output

```
BenchmarkCancel_Atomic_Done_Direct-24    1000000000    0.34 ns/op    0 B/op    0 allocs/op
```

- `-24`: Number of CPUs used (GOMAXPROCS)
- `1000000000`: Iterations run
- `0.34 ns/op`: Time per operation
- `0 B/op`: Bytes allocated per operation
- `0 allocs/op`: Heap allocations per operation

### Expected Variance

- **Good:** < 2% variance
- **Acceptable:** 2-5% variance
- **Investigate:** > 5% variance

High variance causes and mitigations:

| Cause | Mitigation |
|-------|------------|
| Background processes | `nice -n -20`, close browsers/IDEs |
| CPU frequency scaling | Set governor to `performance` |
| Thermal throttling | Let CPU cool between runs |
| Memory pressure | Close memory-heavy apps |
| Goroutine scheduling | `GOMAXPROCS=1` |
| OS scheduler preemption | `taskset -c 0` + `nice -n -20` |
| Hyperthreading noise | Pin to physical core |

### Sanity Checks

1. **Allocations should be 0** for hot-path operations
2. **Relative ordering should be stable** across runs
3. **TSC results may vary** with CPU frequency changes

## CLI Tools

### cmd/context

Compare context cancellation checking:

```bash
go run ./cmd/context -n 10000000
```

### cmd/channel

Compare queue implementations:

```bash
go run ./cmd/channel -n 10000000 -size 1024
```

### cmd/ticker

Compare ticker implementations:

```bash
go run ./cmd/ticker -n 10000000
```

### cmd/context-ticker

Combined benchmark (most realistic):

```bash
go run ./cmd/context-ticker -n 10000000
```

## Typical Results

Results on AMD Ryzen Threadripper PRO 3945WX:

| Component | Standard | Optimized | Speedup |
|-----------|----------|-----------|---------|
| Cancel check | ~10 ns | ~0.3 ns | **30x** |
| Tick check | ~100 ns | ~6 ns (batch) | **16x** |
| Combined | ~96 ns | ~5 ns | **18x** |

## Caveats

1. **Micro-benchmarks measure one dimension** — Real applications have many factors
2. **Results are hardware-dependent** — Your mileage will vary
3. **go:linkname may break** — `runtime.nanotime` is internal
4. **TSC requires calibration** — Accuracy depends on CPU frequency stability

## Profiling

### CPU Profile

```bash
go test -bench=BenchmarkCancel -cpuprofile=cpu.prof ./internal/cancel
go tool pprof -http=:8080 cpu.prof
```

### Memory Profile

```bash
go test -bench=BenchmarkQueue -memprofile=mem.prof ./internal/queue
go tool pprof -http=:8080 mem.prof
```

### Trace

```bash
go test -bench=BenchmarkCombined -trace=trace.out ./internal/combined
go tool trace trace.out
```
