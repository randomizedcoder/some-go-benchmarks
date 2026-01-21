# some-go-benchmarks

Micro-benchmarks for Go concurrency patterns in **polling hot-loops**.

> ⚠️ **Scope:** These benchmarks apply to polling patterns (with `default:` case) where you check channels millions of times per second. Most Go code uses blocking patterns instead—see [Polling vs Blocking](#polling-vs-blocking-when-do-these-benchmarks-apply) before drawing conclusions.

📖 **New to this repo?** Start with the [Walkthrough](WALKTHROUGH.md) for a guided tour with example outputs.

## Results at a Glance

Measured on AMD Ryzen Threadripper PRO 3945WX, Go 1.25, Linux:

### Isolated Operations

| Operation | Standard | Optimized | Speedup |
|-----------|----------|-----------|---------|
| Cancel check | 8.2 ns | 0.36 ns | **23x** |
| Tick check | 86 ns | 5.6 ns | **15x** |
| Queue push+pop | 37 ns | 36 ns | ~1x |

### Combined Hot-Loop Pattern

```go
for {
    if ctx.Done() { return }      // ← Cancel check
    if ticker.Tick() { flush() }  // ← Tick check
    process(queue.Pop())          // ← Queue op
}
```

| Pattern | Standard | Optimized | Speedup |
|---------|----------|-----------|---------|
| Cancel + Tick | 90 ns | 27 ns | **3.4x** |
| Full loop | 130 ns | 63 ns | **2.1x** |

### Real-World Impact

| Throughput | Standard CPU | Optimized CPU | You Save |
|------------|--------------|---------------|----------|
| 100K ops/sec | 1.3% | 0.6% | 0.7% of a core |
| 1M ops/sec | 13% | 6% | **7% of a core** |
| 10M ops/sec | 130% | 63% | **67% of a core** |

> **TL;DR:** At 10M ops/sec, switching to optimized patterns frees up 2/3 of a CPU core.

---

## The Problem

At the scale of millions of operations per second, idiomatic Go constructs like select on time.Ticker or standard channels introduce significant overhead. These bottlenecks stem from:

- Runtime Scheduling: The cost of parking/unparking goroutines.
- Lock Contention: The centralized timer heap in the Go runtime.
- Channel Internals: The overhead of hchan locking and memory barriers.

Example of code that can hit limits in tight loops:
```go
select {
case <-ctx.Done(): return
case <-dropTicker.C: ...
default:  // Non-blocking: returns immediately if nothing ready
}
```

## Polling vs Blocking: When Do These Benchmarks Apply?

Most Go code **blocks** rather than **polls**. Understanding this distinction is critical for interpreting these benchmarks.

### Blocking (Idiomatic Go)

```go
select {
case <-ctx.Done():
    return
case v := <-ch:
    process(v)
// No default: goroutine parks until something is ready
}
```

- **How it works:** Goroutine yields to scheduler, wakes when a channel is ready
- **CPU usage:** Near zero while waiting
- **Latency:** Adds ~1-5µs scheduler wake-up time
- **When to use:** 99% of Go code—network servers, background workers, most pipelines

### Polling (Hot-Loop)

```go
for {
    select {
    case <-ctx.Done():
        return
    case v := <-ch:
        process(v)
    default:
        // Do other work, check again immediately
    }
}
```

- **How it works:** Goroutine never parks, continuously checks channels
- **CPU usage:** 100% of one core while running
- **Latency:** Sub-microsecond response to channel events
- **When to use:** High-throughput loops, soft real-time, packet processing

### Which World Are You In?

| Your Situation | Pattern | These Benchmarks Apply? |
|----------------|---------|------------------------|
| HTTP server handlers | Blocking | ❌ Scheduler cost dominates |
| Background job worker | Blocking | ❌ Use standard patterns |
| Packet processing at 1M+ pps | Polling | ✅ Check overhead matters |
| Game loop / audio processing | Polling | ✅ Every nanosecond counts |
| Streaming data pipeline | Either | ⚠️ Depends on throughput |

> **Key insight:** In blocking code, the scheduler wake-up cost (~1-5µs) dwarfs the channel check overhead (~20ns). Optimizing the check is pointless. In polling code, you're paying that check cost millions of times per second—that's where these optimizations shine.

## Benchmarked Patterns

This repo benchmarks **polling hot-loop** patterns where check overhead is the bottleneck.

### Isolated Micro-Benchmarks

Measure the raw cost of individual operations:

| Category     | Standard Approach        | Optimized Alternatives                        |
|--------------|--------------------------|-----------------------------------------------|
| Cancellation | `select` on `ctx.Done()` | `atomic.Bool` flag                            |
| Messaging    | Buffered `chan` (SPSC)   | Lock-free Ring Buffer                         |
| Time/Tick    | `time.Ticker` in select  | Batching / Atomic / `nanotime` / TSC assembly |

### Combined Interaction Benchmarks

**The most credible guidance** comes from testing interactions, not isolated micro-costs:

| Benchmark | What It Measures |
|-----------|------------------|
| `context-ticker` | Combined cost of checking cancellation + periodic tick |
| `channel-context` | Message processing with cancellation check per message |
| `full-loop` | Realistic hot loop: receive → process → check cancel → check tick |

> **Why combined matters:** Isolated benchmarks can be misleading. A 10x speedup on context checking means nothing if your loop is bottlenecked on channel receives. The combined benchmarks reveal the *actual* improvement in realistic scenarios.

## High-Performance Alternatives


### Lock-Free Ring Buffer

In place of standard channels, we evaluate lock-free ring buffers for lower-latency communication between goroutines.

→ [github.com/randomizedcoder/go-lock-free-ring](https://github.com/randomizedcoder/go-lock-free-ring)

### Atomic Flags for Cancellation

Instead of polling ctx.Done() in a select block, we use an atomic.Bool updated by a separate watcher goroutine. This replaces a channel receive with a much faster atomic load operation.

### Ticker Alternatives (Under Development)

Standard time.Ticker uses the runtime's central timer heap, which can cause contention in high-performance apps. We are exploring:

- Batch-based counters: Only checking the time every N operations.
- Atomic time-sampling: Using a single global goroutine to update an atomic timestamp.

#### The "Every N" Batch Check

If your loop processes items rapidly, checking the clock on every iteration is expensive. Instead, check the time only once every 1,000 or 10,000 iterations.

```
if count++; count % 1000 == 0 {
    if time.Since(lastTick) >= interval {
        // Run logic
        lastTick = time.Now()
    }
}
```
#### Atomic Global Timestamp

If you have many goroutines that all need a "ticker," don't give them each a time.Ticker. Use one background goroutine that updates a global atomic variable with the current Unix nanoseconds. Your workers can then perform a simple atomic comparison.

#### Busy-Wait "Spin" Ticker

For sub-microsecond precision where CPU usage is less important than latency, you can "spin" on the CPU until a specific runtime.nanotime is reached. This avoids the overhead of the Go scheduler parking and unparking your goroutine.

#### Assembly-based TSC (Time Stamp Counter)

For the lowest possible latency on x86, bypass the OS clock entirely and read the CPU's TSC directly. This is significantly faster than `time.Now()` because it avoids the overhead of the Go runtime and VDSO.

- **Mechanism:** Use a small assembly stub or `unsafe` to call the `RDTSC` instruction.
- **Trade-off:** Requires calibration (mapping cycles to nanoseconds) and can be affected by CPU frequency scaling.

```go
// internal/tick/tsc_amd64.s
TEXT ·rdtsc(SB), NOSPLIT, $0-8
    RDTSC
    SHLQ $32, DX
    ORQ  DX, AX
    MOVQ AX, ret+0(FP)
    RET
```

#### runtime.nanotime (Internal Clock)

The Go runtime has an internal function `nanotime()` that returns a monotonic clock value. It is faster than `time.Now()` because it returns a single `int64` and avoids the overhead of constructing a `time.Time` struct.

- **Mechanism:** Access via `//go:linkname`.
- **Benefit:** Provides a middle ground between standard library safety and raw assembly speed.

```go
//go:linkname nanotime runtime.nanotime
func nanotime() int64
```

## Repo layout

The project layout is:
```
[das@l:~/Downloads/some-go-benchmarks]$ tree
.
├── cmd
│   ├── channel
│   ├── context
│   ├── context-ticker
│   └── ticker
├── internal
├── LICENSE
└── README.md

7 directories, 2 files
```

The internal folder is for small library functions that holds the main code.

The ./cmd/ folder has a main.go implmentations that use the libraries, to demostrate limits.

## How to Run

```bash
# Run all tests
go test ./...

# Run benchmarks with memory stats
go test -bench=. -benchmem ./internal/...

# Run specific benchmark with multiple iterations (recommended for microbenches)
go test -run=^$ -bench=BenchmarkQueue -count=10 ./internal/queue

# Run with race detector (slower, but catches concurrency bugs)
go test -race ./...

# Compare results with benchstat (install: go install golang.org/x/perf/cmd/benchstat@latest)
go test -bench=. -count=10 ./internal/cancel > old.txt
# make changes...
go test -bench=. -count=10 ./internal/cancel > new.txt
benchstat old.txt new.txt
```

## Interpreting Results

Micro-benchmarks measure **one dimension** in **one environment**. Keep these caveats in mind:

| Factor | Impact |
|--------|--------|
| Go version | Runtime internals change between releases |
| CPU architecture | x86 vs ARM, cache sizes, branch prediction |
| `GOMAXPROCS` | Contention patterns vary with parallelism |
| Power management | Turbo boost, frequency scaling affect TSC |
| Thermal state | Sustained load causes thermal throttling |

**Recommendations:**

1. **Use `benchstat`** — Run benchmarks 10+ times and use `benchstat` to get statistically meaningful comparisons
2. **Pin CPU frequency** — For TSC benchmarks: `sudo cpupower frequency-set -g performance`
3. **Isolate cores** — For lowest variance: `taskset -c 0 go test -bench=...`
4. **Test your workload** — These are micro-benchmarks; your mileage will vary in real applications
5. **Profile, don't assume** — Use `go tool pprof` to confirm where time actually goes

> **Remember:** A 10x speedup on a 20ns operation saves 180ns per call. If your loop runs 1M times/second, that's 180ms saved per second. If it runs 1000 times/second, that's 0.18ms—probably not worth the complexity.

## Library Design

The `internal/` package provides minimal, focused implementations for benchmarking. Each sub-package exposes a single interface with two implementations: the standard library approach and the optimized alternative.

### Package Structure

```
internal/
├── cancel/          # Cancellation signaling
│   ├── cancel.go    # Interface definition
│   ├── context.go   # Standard: ctx.Done() via select
│   └── atomic.go    # Optimized: atomic.Bool flag
│
├── queue/           # SPSC message passing
│   ├── queue.go     # Interface definition
│   ├── channel.go   # Standard: buffered channel
│   └── ringbuf.go   # Optimized: lock-free ring buffer
│
└── tick/            # Periodic triggers
    ├── tick.go      # Interface definition
    ├── ticker.go    # Standard: time.Ticker in select
    ├── batch.go     # Optimized: check every N ops
    ├── atomic.go    # Optimized: shared atomic timestamp
    ├── nanotime.go  # Optimized: runtime.nanotime via linkname
    └── tsc_amd64.s  # Optimized: raw RDTSC assembly (x86)
```

### Interfaces

Each package defines a minimal interface that both implementations satisfy:

```go
// internal/cancel/cancel.go
package cancel

// Canceler signals shutdown to workers.
type Canceler interface {
    Done() bool   // Returns true if cancelled
    Cancel()      // Trigger cancellation
}
```

```go
// internal/queue/queue.go
package queue

// Queue is a single-producer single-consumer queue.
type Queue[T any] interface {
    Push(T) bool  // Returns false if full
    Pop() (T, bool)
}
```

```go
// internal/tick/tick.go
package tick

// Ticker signals periodic events.
type Ticker interface {
    Tick() bool   // Returns true if interval elapsed
    Reset()       // Reset without reallocation
    Stop()
}
```

### Constructors

Standard Go convention—return concrete types, accept interfaces:

```go
// Standard implementations
cancel.NewContext(ctx context.Context) *ContextCanceler
queue.NewChannel[T any](size int) *ChannelQueue[T]
tick.NewTicker(interval time.Duration) *StdTicker

// Optimized implementations
cancel.NewAtomic() *AtomicCanceler
queue.NewRingBuffer[T any](size int) *RingBuffer[T]
tick.NewBatch(interval time.Duration, every int) *BatchTicker
tick.NewAtomicTicker(interval time.Duration) *AtomicTicker
tick.NewNanotime(interval time.Duration) *NanotimeTicker
tick.NewTSC(interval, cyclesPerNs float64) *TSCTicker  // x86 only
tick.NewTSCCalibrated(interval time.Duration) *TSCTicker  // auto-calibrates
```

### Benchmark Pattern

Each `cmd/` binary follows the same structure:

```go
func main() {
    // Parse flags for iterations, warmup, etc.

    // Run standard implementation
    std := runBenchmark(standardImpl, iterations)

    // Run optimized implementation
    opt := runBenchmark(optimizedImpl, iterations)

    // Print comparison
    fmt.Printf("Standard: %v\nOptimized: %v\nSpeedup: %.2fx\n",
        std, opt, float64(std)/float64(opt))
}
```

### Design Principles

1. **No abstraction for abstraction's sake**—interfaces exist only because we need to swap implementations
2. **Zero allocations in hot paths**—pre-allocate, reuse, avoid escape to heap
3. **Benchmark-friendly**—implementations expose internals needed for accurate measurement
4. **Copy-paste ready**—each optimized implementation is self-contained for easy extraction