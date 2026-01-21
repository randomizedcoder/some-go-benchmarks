# Benchmarking Walkthrough

This document walks you through running benchmarks and interpreting results.
Your results will vary based on your hardware, but this gives you an idea of what to expect.

## Test System

```
OS:     Linux 6.18.5 (NixOS)
CPU:    AMD Ryzen Threadripper PRO 3945WX 12-Cores
Cores:  12 physical, 24 logical (hyperthreading)
RAM:    128 GB
Go:     go1.25.5 linux/amd64
```

---

## Step 1: Verify Installation

First, make sure everything builds and tests pass:

```bash
$ go build ./...
$ go test ./...
```

**Expected output:**

```
ok      github.com/randomizedcoder/some-go-benchmarks/internal/cancel   0.003s
ok      github.com/randomizedcoder/some-go-benchmarks/internal/combined 0.002s
ok      github.com/randomizedcoder/some-go-benchmarks/internal/queue    0.004s
ok      github.com/randomizedcoder/some-go-benchmarks/internal/tick     0.735s
```

### Using the Makefile

The project includes a Makefile with convenient targets:

```bash
$ make help
```

**Output:**

```
Available targets:

Build & Test:
  build          - Build all packages
  test           - Run all tests
  race           - Run tests with race detector
  lint           - Run golangci-lint
  check          - Run build, test, and race

All Benchmarks:
  bench          - Run all benchmarks with memory stats
  bench-count    - Run benchmarks 10 times (for variance)
  bench-variance - Run benchmarks and save for benchstat
  bench-race     - Run benchmarks with race detector

Category Benchmarks:
  bench-cancel   - Cancel check: context vs atomic
  bench-tick     - Tick check: ticker implementations
  bench-queue    - Queue: single goroutine push+pop
  bench-pipeline - Pipeline: 2-goroutine SPSC producer/consumer
  bench-mpsc     - MPSC: N producers -> 1 consumer (channel contention)
  bench-lfr      - go-lock-free-ring comparison (SPSC vs MPSC)
  bench-combined - Combined loop: cancel + tick + queue

Cleanup:
  clean          - Remove generated files
```

**Quick Start:**

```bash
# Run all benchmarks
$ make bench

# Run specific category
$ make bench-lfr      # go-lock-free-ring comparison
$ make bench-mpsc     # Channel contention with multiple producers
$ make bench-pipeline # 2-goroutine producer/consumer
```

---

## Step 2: Run Basic Benchmarks

### Cancel Package

```bash
$ go test -bench=. -benchmem ./internal/cancel
```

**Output:**

```
goos: linux
goarch: amd64
pkg: github.com/randomizedcoder/some-go-benchmarks/internal/cancel
cpu: AMD Ryzen Threadripper PRO 3945WX 12-Cores
BenchmarkCancel_Context_Done_Direct-24          138030020                8.232 ns/op           0 B/op          0 allocs/op
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.3575 ns/op          0 B/op          0 allocs/op
BenchmarkCancel_Context_Done_Interface-24       143021458                8.193 ns/op           0 B/op          0 allocs/op
BenchmarkCancel_Atomic_Done_Interface-24        1000000000               0.3751 ns/op          0 B/op          0 allocs/op
BenchmarkCancel_Context_Done_Parallel-24        1000000000               0.6508 ns/op          0 B/op          0 allocs/op
BenchmarkCancel_Atomic_Done_Parallel-24         1000000000               0.07654 ns/op         0 B/op          0 allocs/op
BenchmarkCancel_Atomic_Reset-24                 279049110                4.501 ns/op           0 B/op          0 allocs/op
PASS
ok      github.com/randomizedcoder/some-go-benchmarks/internal/cancel   7.361s
```

**How to read this:**

| Column | Meaning |
|--------|---------|
| `-24` | Using 24 CPU threads (GOMAXPROCS) |
| `138030020` | Number of iterations run |
| `8.232 ns/op` | 8.232 nanoseconds per operation |
| `0 B/op` | Zero bytes allocated per operation |
| `0 allocs/op` | Zero heap allocations per operation |

**Key insight:** Atomic is **23x faster** than Context (0.36 ns vs 8.23 ns)

---

### Tick Package

```bash
$ go test -bench=. -benchmem ./internal/tick
```

**Output:**

```
BenchmarkTick_Std_Direct-24             13369196                86.24 ns/op           0 B/op          0 allocs/op
BenchmarkTick_Batch_Direct-24           209211277                5.627 ns/op          0 B/op          0 allocs/op
BenchmarkTick_Atomic_Direct-24          41821100                25.71 ns/op           0 B/op          0 allocs/op
BenchmarkTick_TSC_Direct-24             131311492                9.436 ns/op          0 B/op          0 allocs/op
```

**Performance ranking:**

| Implementation | ns/op | Speedup vs Std |
|----------------|-------|----------------|
| StdTicker | 86.24 | 1x (baseline) |
| AtomicTicker | 25.71 | 3.4x |
| TSCTicker | 9.44 | 9.1x |
| BatchTicker | 5.63 | **15.3x** |

---

### Combined Benchmarks (Most Realistic)

```bash
$ go test -bench=. -benchmem ./internal/combined
```

**Output:**

```
BenchmarkCombined_CancelTick_Standard-24        13146752                90.10 ns/op            0 B/op          0 allocs/op
BenchmarkCombined_CancelTick_Optimized-24       45594999                26.75 ns/op            0 B/op          0 allocs/op
BenchmarkCombined_FullLoop_Standard-24           9150345               130.2 ns/op             0 B/op          0 allocs/op
BenchmarkCombined_FullLoop_Optimized-24         19513278                62.86 ns/op            0 B/op          0 allocs/op
```

**Key insight:** Combined optimizations give **2.1x speedup** on the full loop (130 ns → 63 ns)

---

### Queue Benchmarks: Goroutine Patterns

Queue performance varies dramatically based on goroutine topology.

#### Single Goroutine (No Contention)

```bash
$ go test -bench=BenchmarkQueue -benchmem ./internal/queue
```

**Output:**

```
goos: linux
goarch: amd64
pkg: github.com/randomizedcoder/some-go-benchmarks/internal/queue
cpu: AMD Ryzen Threadripper PRO 3945WX 12-Cores
BenchmarkQueue_Channel_PushPop_Direct-24           30932498                38.96 ns/op            0 B/op          0 allocs/op
BenchmarkQueue_RingBuffer_PushPop_Direct-24        32920832                35.89 ns/op            0 B/op          0 allocs/op
BenchmarkQueue_Channel_PushPop_Interface-24        27947314                43.26 ns/op            0 B/op          0 allocs/op
BenchmarkQueue_RingBuffer_PushPop_Interface-24     30313048                40.37 ns/op            0 B/op          0 allocs/op
```

> **Note:** The in-repo RingBuffer has SPSC guards that add overhead. An unguarded ring buffer achieves ~9.5 ns/op.

#### SPSC: 1 Producer → 1 Consumer (2 Goroutines)

This is the classic producer/consumer pattern—the most common Go concurrency pattern:

```bash
$ go test -bench=BenchmarkPipeline -benchmem ./internal/combined
```

**Output:**

```
goos: linux
goarch: amd64
pkg: github.com/randomizedcoder/some-go-benchmarks/internal/combined
cpu: AMD Ryzen Threadripper PRO 3945WX 12-Cores
BenchmarkPipeline_Channel-24             7858700               127.9 ns/op            0 B/op           0 allocs/op
BenchmarkPipeline_RingBuffer-24         11740012               146.8 ns/op            0 B/op           0 allocs/op
```

> **Key insight:** The guarded RingBuffer is *slower* than channels due to SPSC guard overhead. An unguarded lock-free ring buffer achieves ~39 ns/op (**3.3x faster** than channels).

#### MPSC: N Producers → 1 Consumer (Channel Lock Contention)

Multiple goroutines sending to one consumer shows channel lock contention:

```bash
$ go test -bench=BenchmarkMPSC -benchmem ./internal/combined
```

**Output:**

```
goos: linux
goarch: amd64
pkg: github.com/randomizedcoder/some-go-benchmarks/internal/combined
cpu: AMD Ryzen Threadripper PRO 3945WX 12-Cores
BenchmarkMPSC_Channel_2Producers-24       180842              5922 ns/op              0 B/op           0 allocs/op
BenchmarkMPSC_Channel_4Producers-24       119090             26351 ns/op              0 B/op           0 allocs/op
BenchmarkMPSC_Channel_8Producers-24       171520             49074 ns/op              0 B/op           0 allocs/op
```

**Lock contention scaling:**

| Producers | Latency | vs 1 Producer |
|-----------|---------|---------------|
| 1 (SPSC) | 128 ns | baseline |
| 2 | 5.9 µs | **46x** slower |
| 4 | 26 µs | **200x** slower |
| 8 | 49 µs | **380x** slower |

> **Key insight:** Channel lock contention scales poorly. For high-throughput fan-in patterns, use go-lock-free-ring.

#### go-lock-free-ring Comparison

The [go-lock-free-ring](https://github.com/randomizedcoder/go-lock-free-ring) library provides a sharded MPSC ring buffer. Let's compare:

```bash
$ go test -bench=BenchmarkLFR -benchmem ./internal/combined
```

**Output:**

```
goos: linux
goarch: amd64
pkg: github.com/randomizedcoder/some-go-benchmarks/internal/combined
cpu: AMD Ryzen Threadripper PRO 3945WX 12-Cores
BenchmarkLFR_SPSC_Channel-24                     4372419               247.8 ns/op             0 B/op          0 allocs/op
BenchmarkLFR_SPSC_OurRing-24                    33405556                36.53 ns/op            0 B/op          0 allocs/op
BenchmarkLFR_SPSC_ShardedRing1-24               10212240               114.1 ns/op             8 B/op          1 allocs/op
BenchmarkLFR_MPSC_Channel_4P-24                    85386             35337 ns/op               0 B/op          0 allocs/op
BenchmarkLFR_MPSC_ShardedRing_4P_4S-24           2179492               539.4 ns/op           412 B/op         51 allocs/op
BenchmarkLFR_MPSC_Channel_8P-24                    58347             47067 ns/op               1 B/op          0 allocs/op
BenchmarkLFR_MPSC_ShardedRing_8P_8S-24           2596642               464.0 ns/op           412 B/op         51 allocs/op
```

**SPSC Comparison (1 Producer → 1 Consumer):**

These are **cross-goroutine** benchmarks (separate producer/consumer goroutines):

| Implementation | Latency | Allocs | Speedup |
|----------------|---------|--------|---------|
| Channel | 248 ns | 0 | baseline |
| go-lock-free-ring (1 shard) | 114 ns | 1 | 2.2x |
| **Our SPSC Ring** | **36.5 ns** | **0** | **6.8x** |

> **Note:** go-lock-free-ring's native `BenchmarkProducerConsumer` shows 31 ns/op, but that's in the **same goroutine**. Cross-goroutine polling adds ~80ns coordination overhead.

**Why Our SPSC Ring is Faster in Cross-Goroutine Tests:**

1. **CAS vs Store**: go-lock-free-ring uses `CompareAndSwap` to safely handle multiple producers (3-10x slower than simple Store)
2. **Sequence numbers**: go-lock-free-ring uses per-slot sequence numbers to prevent race conditions (extra atomic ops)
3. **Boxing**: go-lock-free-ring uses `any` type causing allocations; our ring uses generics (zero allocs)

> **Tradeoff**: Our ring is faster because it makes dangerous assumptions (single producer, x86 memory model). go-lock-free-ring is slower because it's provably race-free.

**MPSC Comparison (N Producers → 1 Consumer):**

| Producers | Channel | go-lock-free-ring | Speedup |
|-----------|---------|-------------------|---------|
| 4 | 35.3 µs | 539 ns | **65x** |
| 8 | 47.1 µs | 464 ns | **101x** |

> **Key insight:** go-lock-free-ring's sharded design eliminates lock contention, providing **65-101x** speedup over channels for multi-producer scenarios!

**Choosing the Right Queue:**

| Your Pattern | Best Choice | Why |
|--------------|-------------|-----|
| 1 producer, 1 consumer | Our SPSC Ring | Fastest (36.5 ns), zero allocs |
| N producers, 1 consumer | go-lock-free-ring | Sharding eliminates contention |
| Simple/infrequent | Channel | Simplicity over speed |

---

## Step 3: Use CLI Tools

The CLI tools provide easier-to-read output with throughput analysis.

### Context Cancellation Comparison

```bash
$ go run ./cmd/context -n 5000000
```

**Output:**

```
Benchmarking cancellation check (5000000 iterations)
─────────────────────────────────────────────────

Results:
  Context:  43.74395ms (8.75 ns/op)
  Atomic:   1.640922ms (0.33 ns/op)

  Speedup:  26.66x

Throughput (theoretical max):
  Context:  114.30 M ops/sec
  Atomic:   3047.07 M ops/sec
```

### Combined Cancel + Tick (Most Realistic)

```bash
$ go run ./cmd/context-ticker -n 5000000
```

**Output:**

```
Benchmarking combined cancel+tick check (5000000 iterations)
─────────────────────────────────────────────────────────

This simulates a hot loop that checks for cancellation
and periodic timing on every iteration:

  for {
      if cancel.Done() { return }
      if ticker.Tick() { doPeriodicWork() }
      processItem()
  }

Results:
─────────────────────────────────────────────────────────
  Standard (ctx + time.Ticker):
    Total: 465.769925ms, Per-op: 93.15 ns

  Optimized (atomic + AtomicTicker):
    Total: 134.594392ms, Per-op: 26.92 ns
    Speedup: 3.46x

  Ultra (atomic + BatchTicker):
    Total: 25.06717ms, Per-op: 5.01 ns
    Speedup: 18.58x

Impact Analysis:
─────────────────────────────────────────────────────────
  Savings per iteration: 66.24 ns

  At 100K ops/sec: save 6.62 ms/sec (0.66% of 1 core)
  At 1000K ops/sec: save 66.24 ms/sec (6.62% of 1 core)
  At 10000K ops/sec: save 662.35 ms/sec (66.24% of 1 core)
```

**What this tells you:**
- At 1M operations/second, you save **66ms of CPU time per second**
- At 10M operations/second, you save **662ms** — that's 66% of a CPU core!

---

## Step 4: Variance Analysis

Run benchmarks multiple times to check consistency:

```bash
$ go test -bench=BenchmarkCancel_Atomic_Done_Direct -count=5 ./internal/cancel
```

**Output:**

```
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.3794 ns/op
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.4376 ns/op
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.3601 ns/op
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.3526 ns/op
BenchmarkCancel_Atomic_Done_Direct-24           1000000000               0.3450 ns/op
```

**Analysis:**
- Range: 0.345 - 0.438 ns/op
- Variance: ~27% (the 0.44 is an outlier)
- Most results cluster around 0.35-0.38 ns

**Tip:** Use `benchstat` for statistical analysis:

```bash
$ go install golang.org/x/perf/cmd/benchstat@latest
$ go test -bench=. -count=10 ./internal/cancel > results.txt
$ benchstat results.txt
```

---

## Step 5: Environment Tuning

### With GOMAXPROCS=1

Reduce Go scheduler noise by using a single thread:

```bash
$ GOMAXPROCS=1 go test -bench=BenchmarkCancel_Atomic_Done_Direct -benchmem ./internal/cancel
```

**Output:**

```
BenchmarkCancel_Atomic_Done_Direct      1000000000               0.4111 ns/op          0 B/op          0 allocs/op
```

Notice: `-24` suffix is now missing (single-threaded).

### With CPU Pinning

```bash
$ taskset -c 0 GOMAXPROCS=1 go test -bench=BenchmarkCancel_Atomic_Done_Direct ./internal/cancel
```

### With High Priority

```bash
$ sudo nice -n -20 go test -bench=. ./internal/cancel
```

### Maximum Isolation

```bash
$ sudo nice -n -20 taskset -c 0 GOMAXPROCS=1 go test -bench=. ./internal/cancel
```

---

## Step 6: Understanding the Results

### Summary Table

| Component | Standard | Optimized | Speedup |
|-----------|----------|-----------|---------|
| Cancel check | 8.2 ns | 0.36 ns | **23x** |
| Tick check | 86 ns | 5.6 ns (batch) | **15x** |
| Queue SPSC (2 goroutines) | 248 ns | 36.5 ns | **6.8x** |
| Queue MPSC (8 producers) | 47 µs | 464 ns | **101x** |
| Combined loop | 130 ns | 63 ns | **2.1x** |

### When Do These Optimizations Matter?

| Operations/sec | Standard CPU | Optimized CPU | Savings |
|----------------|--------------|---------------|---------|
| 100K | 0.9% | 0.3% | 0.6% |
| 1M | 9% | 3% | 6% |
| 10M | 90% | 30% | **60%** |

**Rule of thumb:** If you're doing >1M operations/second in a hot loop, these optimizations matter significantly.

---

## Step 7: Profiling (Optional)

### CPU Profile

```bash
$ go test -bench=BenchmarkCombined -cpuprofile=cpu.prof ./internal/combined
$ go tool pprof -http=:8080 cpu.prof
```

Opens a web UI showing where time is spent.

### Memory Profile

```bash
$ go test -bench=BenchmarkQueue -memprofile=mem.prof ./internal/queue
$ go tool pprof -http=:8080 mem.prof
```

All benchmarks should show 0 allocations.

---

## Common Issues

### High Variance

**Symptom:** Results vary by >10% between runs.

**Causes:**
- Background processes (browser, IDE)
- CPU frequency scaling
- Thermal throttling

**Fix:**
```bash
# Kill background apps, then:
sudo cpupower frequency-set -g performance
sudo nice -n -20 taskset -c 0 GOMAXPROCS=1 go test -bench=. ./internal/...
```

### Unexpected Results

**Symptom:** Optimized version is slower than standard.

**Possible causes:**
1. **SPSC guards:** RingBuffer has safety checks that add overhead
2. **Warm-up:** First run may include JIT/cache warming
3. **Measurement noise:** Run with `-count=10` and use benchstat

---

## Next Steps

1. **Read the code:** Look at `internal/cancel/atomic.go` to see how simple the optimization is
2. **Try in your code:** Replace `ctx.Done()` checks with `AtomicCanceler`
3. **Measure your application:** Profile to see if these hot paths are actually your bottleneck
4. **Don't over-optimize:** If you're not doing millions of ops/sec, standard patterns are fine

---

## Quick Reference

```bash
# Run all benchmarks
make bench

# Run specific package
go test -bench=. ./internal/cancel

# Multiple runs for variance
go test -bench=. -count=10 ./internal/... > results.txt

# Compare with benchstat
benchstat results.txt

# CLI tools
go run ./cmd/context -n 10000000
go run ./cmd/ticker -n 10000000
go run ./cmd/context-ticker -n 10000000
go run ./cmd/channel -n 10000000

# Maximum isolation
sudo nice -n -20 taskset -c 0 GOMAXPROCS=1 go test -bench=. ./internal/...
```
