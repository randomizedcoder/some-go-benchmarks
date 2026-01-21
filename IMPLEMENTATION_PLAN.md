# Implementation Plan

This document outlines the phased approach to implementing the benchmark libraries and command-line tools.

## Scope: Polling Hot-Loops Only

These benchmarks target **polling patterns** (with `default:` case), not blocking patterns.

| Pattern | This Repo? | Why |
|---------|------------|-----|
| Polling hot-loop | ✅ Yes | Check overhead is the bottleneck |
| Blocking select | ❌ No | Scheduler wake-up (~1-5µs) dominates |

**Target use cases:** Packet processing, game loops, audio pipelines, soft real-time systems—anywhere you're processing millions of events per second and can't afford to park goroutines.

## Overview

| Phase | Focus | Deliverables |
|-------|-------|--------------|
| 1 | Project Setup | `go.mod`, directory structure, CI config |
| 2 | Core Libraries | `internal/cancel`, `internal/queue`, `internal/tick` |
| 2.5 | Portability | Build tags, CI matrix, Go version testing |
| 3 | Unit Tests | Correctness tests + contract violation tests |
| 4 | Benchmark Tests | `_bench_test.go` + methodology validation |
| 5 | CLI Tools | `cmd/` binaries demonstrating real-world usage |
| 6 | Validation | Race detection, profiling, documentation |

## Risk Mitigation Summary

| Risk | Mitigation |
|------|------------|
| Benchmark methodology | `-count=10`, variance checks, sink variables, environment locking |
| Correctness/contracts | SPSC violation tests, debug vs release modes |
| Portability | CI matrix (amd64/arm64), multiple Go versions, safe defaults |
| Code duplication | Consolidate `AtomicTicker`/`NanotimeTicker` early |

---

## Phase 1: Project Setup

### Tasks

1. Initialize Go module
   ```bash
   go mod init github.com/randomizedcoder/some-go-benchmarks
   ```

2. Create directory structure
   ```
   internal/
   ├── cancel/
   ├── queue/
   └── tick/
   ```

3. Vendor the lock-free ring buffer dependency
   ```bash
   # From local source
   cp -r ~/Downloads/go-lock-free-ring ./vendor/
   # Or add as module dependency
   go get github.com/randomizedcoder/go-lock-free-ring
   ```

4. Create `Makefile` with standard targets
   ```makefile
   .PHONY: test bench race lint

   test:
   	go test ./...

   bench:
   	go test -bench=. -benchmem ./...

   race:
   	go test -race ./...

   lint:
   	golangci-lint run
   ```

### Exit Criteria
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` runs (even with no tests)

---

## Phase 2: Core Libraries

Implement each package in order of dependency (none depend on each other, so order is flexible).

### 2.1 `internal/cancel`

**Files:**
| File | Purpose |
|------|---------|
| `cancel.go` | Interface definition |
| `context.go` | Standard: wraps `context.Context` |
| `atomic.go` | Optimized: `atomic.Bool` flag |

**Implementation:**

```go
// cancel.go
package cancel

// Canceler provides cancellation signaling.
type Canceler interface {
    Done() bool
    Cancel()
}
```

```go
// context.go
package cancel

import "context"

type ContextCanceler struct {
    ctx    context.Context
    cancel context.CancelFunc
}

func NewContext(parent context.Context) *ContextCanceler {
    ctx, cancel := context.WithCancel(parent)
    return &ContextCanceler{ctx: ctx, cancel: cancel}
}

func (c *ContextCanceler) Done() bool {
    select {
    case <-c.ctx.Done():
        return true
    default:
        return false
    }
}

func (c *ContextCanceler) Cancel() {
    c.cancel()
}
```

```go
// atomic.go
package cancel

import "sync/atomic"

type AtomicCanceler struct {
    done atomic.Bool
}

func NewAtomic() *AtomicCanceler {
    return &AtomicCanceler{}
}

func (a *AtomicCanceler) Done() bool {
    return a.done.Load()
}

func (a *AtomicCanceler) Cancel() {
    a.done.Store(true)
}
```

---

### 2.2 `internal/queue`

**Files:**
| File | Purpose |
|------|---------|
| `queue.go` | Interface definition |
| `channel.go` | Standard: buffered channel |
| `ringbuf.go` | Optimized: lock-free ring buffer wrapper |

**Implementation:**

```go
// queue.go
package queue

// Queue is a single-producer single-consumer queue.
type Queue[T any] interface {
    Push(T) bool
    Pop() (T, bool)
}
```

```go
// channel.go
package queue

type ChannelQueue[T any] struct {
    ch chan T
}

func NewChannel[T any](size int) *ChannelQueue[T] {
    return &ChannelQueue[T]{ch: make(chan T, size)}
}

func (q *ChannelQueue[T]) Push(v T) bool {
    select {
    case q.ch <- v:
        return true
    default:
        return false
    }
}

func (q *ChannelQueue[T]) Pop() (T, bool) {
    select {
    case v := <-q.ch:
        return v, true
    default:
        var zero T
        return zero, false
    }
}
```

```go
// ringbuf.go
package queue

import (
    "sync/atomic"

    ring "github.com/randomizedcoder/go-lock-free-ring"
)

// RingBuffer is a lock-free SPSC (Single-Producer Single-Consumer) queue.
//
// WARNING: This queue is NOT safe for multiple producers or multiple consumers.
// Using it incorrectly will cause data races and undefined behavior.
// The debug guards below help catch misuse during development.
type RingBuffer[T any] struct {
    ring       *ring.Ring[T]
    pushActive atomic.Uint32 // SPSC guard: detects concurrent Push
    popActive  atomic.Uint32 // SPSC guard: detects concurrent Pop
}

func NewRingBuffer[T any](size int) *RingBuffer[T] {
    return &RingBuffer[T]{ring: ring.New[T](size)}
}

func (r *RingBuffer[T]) Push(v T) bool {
    // SPSC guard: panic if concurrent Push detected
    if !r.pushActive.CompareAndSwap(0, 1) {
        panic("queue: concurrent Push on SPSC RingBuffer")
    }
    defer r.pushActive.Store(0)

    return r.ring.Write(v)
}

func (r *RingBuffer[T]) Pop() (T, bool) {
    // SPSC guard: panic if concurrent Pop detected
    if !r.popActive.CompareAndSwap(0, 1) {
        panic("queue: concurrent Pop on SPSC RingBuffer")
    }
    defer r.popActive.Store(0)

    return r.ring.Read()
}
```

> **SPSC Contract:**
> - Single Producer: Only ONE goroutine may call `Push()`
> - Single Consumer: Only ONE goroutine may call `Pop()`
> - The atomic guards add ~1-2ns overhead but catch misuse early
> - For production without guards, use build tags: `//go:build !debug`

---

### 2.3 `internal/tick`

**Files:**
| File | Purpose |
|------|---------|
| `tick.go` | Interface definition |
| `ticker.go` | Standard: `time.Ticker` wrapper |
| `batch.go` | Optimized: check every N operations |
| `atomic.go` | Optimized: `nanotime` + atomic (declares linkname) |
| `nanotime.go` | Optimized: alternative nanotime ticker |
| `tsc_amd64.go` | Optimized: TSC wrapper + calibration |
| `tsc_amd64.s` | Assembly: raw RDTSC instruction |

**Implementation:**

```go
// tick.go
package tick

// Ticker signals when an interval has elapsed.
type Ticker interface {
    Tick() bool   // Returns true if interval elapsed
    Reset()       // Reset without reallocation (for reuse in hot paths)
    Stop()        // Release resources
}
```

```go
// ticker.go
package tick

import "time"

type StdTicker struct {
    ticker   *time.Ticker
    interval time.Duration
}

func NewTicker(interval time.Duration) *StdTicker {
    return &StdTicker{
        ticker:   time.NewTicker(interval),
        interval: interval,
    }
}

func (t *StdTicker) Tick() bool {
    select {
    case <-t.ticker.C:
        return true
    default:
        return false
    }
}

func (t *StdTicker) Reset() {
    t.ticker.Reset(t.interval)
}

func (t *StdTicker) Stop() {
    t.ticker.Stop()
}
```

```go
// batch.go
package tick

import "time"

type BatchTicker struct {
    interval time.Duration
    every    int
    count    int
    lastTick time.Time
}

func NewBatch(interval time.Duration, every int) *BatchTicker {
    return &BatchTicker{
        interval: interval,
        every:    every,
        lastTick: time.Now(),
    }
}

func (b *BatchTicker) Tick() bool {
    b.count++
    if b.count%b.every != 0 {
        return false
    }
    now := time.Now()
    if now.Sub(b.lastTick) >= b.interval {
        b.lastTick = now
        return true
    }
    return false
}

func (b *BatchTicker) Reset() {
    b.count = 0
    b.lastTick = time.Now()
}

func (b *BatchTicker) Stop() {}
```

```go
// atomic.go
package tick

import (
    "sync/atomic"
    "time"
    _ "unsafe" // for go:linkname
)

//go:linkname nanotime runtime.nanotime
func nanotime() int64

type AtomicTicker struct {
    interval int64 // nanoseconds
    lastTick atomic.Int64
}

func NewAtomicTicker(interval time.Duration) *AtomicTicker {
    t := &AtomicTicker{
        interval: int64(interval),
    }
    t.lastTick.Store(nanotime())
    return t
}

func (a *AtomicTicker) Tick() bool {
    now := nanotime()
    last := a.lastTick.Load()
    if now-last >= a.interval {
        // CAS to prevent multiple triggers
        if a.lastTick.CompareAndSwap(last, now) {
            return true
        }
    }
    return false
}

func (a *AtomicTicker) Reset() {
    a.lastTick.Store(nanotime())
}

func (a *AtomicTicker) Stop() {}
```

> **Note:** `AtomicTicker` now uses `runtime.nanotime` instead of `time.Now().UnixNano()`.
> - `UnixNano()` is wall-clock time and can jump during NTP syncs
> - `nanotime()` is monotonic and avoids VDSO overhead
> - This makes `AtomicTicker` and `NanotimeTicker` functionally identical—consider consolidating

```go
// nanotime.go
package tick

import (
    "sync/atomic"
    "time"
    _ "unsafe" // for go:linkname
)

// Note: nanotime is declared in atomic.go via go:linkname

type NanotimeTicker struct {
    interval int64
    lastTick atomic.Int64
}

func NewNanotime(interval time.Duration) *NanotimeTicker {
    t := &NanotimeTicker{interval: int64(interval)}
    t.lastTick.Store(nanotime())
    return t
}

func (n *NanotimeTicker) Tick() bool {
    now := nanotime()
    last := n.lastTick.Load()
    if now-last >= n.interval {
        if n.lastTick.CompareAndSwap(last, now) {
            return true
        }
    }
    return false
}

func (n *NanotimeTicker) Reset() {
    n.lastTick.Store(nanotime())
}

func (n *NanotimeTicker) Stop() {}
```

```asm
// tsc_amd64.s
#include "textflag.h"

// func rdtsc() uint64
TEXT ·rdtsc(SB), NOSPLIT, $0-8
    RDTSC
    SHLQ $32, DX
    ORQ  DX, AX
    MOVQ AX, ret+0(FP)
    RET
```

```go
// tsc_amd64.go
//go:build amd64

package tick

import (
    "sync/atomic"
    "time"
)

func rdtsc() uint64 // implemented in tsc_amd64.s

// CalibrateTSC measures CPU cycles per nanosecond.
// Call once at startup; result varies with CPU frequency scaling.
func CalibrateTSC() float64 {
    // Warm up
    rdtsc()

    start := rdtsc()
    t1 := time.Now()
    time.Sleep(10 * time.Millisecond)
    end := rdtsc()
    t2 := time.Now()

    cycles := float64(end - start)
    nanos := float64(t2.Sub(t1).Nanoseconds())
    return cycles / nanos
}

type TSCTicker struct {
    intervalCycles uint64
    lastTick       atomic.Uint64
}

// NewTSC creates a TSC-based ticker with explicit cycles/ns ratio.
func NewTSC(interval time.Duration, cyclesPerNs float64) *TSCTicker {
    t := &TSCTicker{
        intervalCycles: uint64(float64(interval.Nanoseconds()) * cyclesPerNs),
    }
    t.lastTick.Store(rdtsc())
    return t
}

// NewTSCCalibrated creates a TSC ticker with auto-calibration.
// Blocks for ~10ms during calibration.
func NewTSCCalibrated(interval time.Duration) *TSCTicker {
    return NewTSC(interval, CalibrateTSC())
}

func (t *TSCTicker) Tick() bool {
    now := rdtsc()
    last := t.lastTick.Load()
    if now-last >= t.intervalCycles {
        if t.lastTick.CompareAndSwap(last, now) {
            return true
        }
    }
    return false
}

func (t *TSCTicker) Reset() {
    t.lastTick.Store(rdtsc())
}

func (t *TSCTicker) Stop() {}
```

> **TSC Considerations:**
> - CPU frequency scaling (Turbo Boost, SpeedStep) affects TSC rate
> - `CalibrateTSC()` provides a point-in-time measurement
> - For highest accuracy, pin to a single CPU core and disable frequency scaling
> - On invariant TSC CPUs (most modern x86), the TSC runs at constant rate regardless of frequency

### Exit Criteria
- [ ] All files compile: `go build ./internal/...`
- [ ] No lint errors: `golangci-lint run ./internal/...`

### Design Decision: Consolidate Nanotime Tickers

`AtomicTicker` and `NanotimeTicker` are now functionally identical (both use `runtime.nanotime`). **Consolidate early** to reduce duplicate code paths and benchmark bugs:

```go
// Keep only AtomicTicker (or rename to NanotimeTicker)
// Delete the duplicate implementation
```

---

## Phase 2.5: Portability & Build Safety

### Goals

Ensure the repo builds and runs correctly across:
- Architectures: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Go versions: 1.21, 1.22, 1.23 (latest)

### Build Tags for Safe Defaults

TSC and `go:linkname` are fragile. Structure code so the **default build always works**:

```
internal/tick/
├── tick.go           # Interface (always builds)
├── ticker.go         # StdLib (always builds)
├── batch.go          # Pure Go (always builds)
├── atomic.go         # nanotime via linkname (needs unsafe import)
├── atomic_safe.go    # Fallback if linkname breaks (build tag)
├── tsc_amd64.go      # TSC (only amd64)
├── tsc_amd64.s       # Assembly (only amd64)
└── tsc_stub.go       # No-op stub for other archs
```

**Build tag pattern:**

```go
// tsc_amd64.go
//go:build amd64

package tick
// ... TSC implementation
```

```go
// tsc_stub.go
//go:build !amd64

package tick

import "errors"

var ErrTSCNotSupported = errors.New("TSC ticker requires amd64")

func NewTSC(interval time.Duration, cyclesPerNs float64) (*TSCTicker, error) {
    return nil, ErrTSCNotSupported
}

func NewTSCCalibrated(interval time.Duration) (*TSCTicker, error) {
    return nil, ErrTSCNotSupported
}
```

### go:linkname Fragility

`runtime.nanotime` is an internal function. It may change or be removed. Add a fallback:

```go
// atomic_safe.go
//go:build go_safe || (!amd64 && !arm64)

package tick

import "time"

// Fallback: use time.Now().UnixNano() if linkname is unavailable
func nanotime() int64 {
    return time.Now().UnixNano()
}
```

### CI Matrix (GitHub Actions)

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        go-version: ['1.21', '1.22', '1.23']
        os: [ubuntu-latest, macos-latest]
        include:
          - os: ubuntu-latest
            arch: amd64
          - os: macos-latest
            arch: arm64

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}

      - name: Build
        run: go build ./...

      - name: Test
        run: go test -race ./...

      - name: Test with safe build tag
        run: go test -tags=go_safe ./...

      - name: Benchmark (quick sanity check)
        run: go test -bench=. -benchtime=100ms ./internal/...
```

### Exit Criteria
- [ ] `go build ./...` succeeds on amd64 and arm64
- [ ] `go test ./...` passes on all CI matrix combinations
- [ ] `go build -tags=go_safe ./...` works without linkname

---

## Phase 3: Unit Tests

Each package gets a `_test.go` file verifying correctness.

### Test Strategy

| Package | Test Focus |
|---------|------------|
| `cancel` | Verify `Done()` returns false before cancel, true after |
| `queue` | Verify FIFO ordering, full/empty behavior |
| `tick` | Verify tick fires after interval, not before |

### 3.1 `internal/cancel/cancel_test.go`

```go
package cancel_test

import (
    "context"
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

func TestContextCanceler(t *testing.T) {
    c := cancel.NewContext(context.Background())

    if c.Done() {
        t.Error("expected Done() = false before Cancel()")
    }

    c.Cancel()

    if !c.Done() {
        t.Error("expected Done() = true after Cancel()")
    }
}

func TestAtomicCanceler(t *testing.T) {
    c := cancel.NewAtomic()

    if c.Done() {
        t.Error("expected Done() = false before Cancel()")
    }

    c.Cancel()

    if !c.Done() {
        t.Error("expected Done() = true after Cancel()")
    }
}
```

### 3.2 `internal/queue/queue_test.go`

```go
package queue_test

import (
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

func testQueue[T comparable](t *testing.T, q queue.Queue[T], val T) {
    // Empty queue returns false
    if _, ok := q.Pop(); ok {
        t.Error("expected Pop() = false on empty queue")
    }

    // Push succeeds
    if !q.Push(val) {
        t.Error("expected Push() = true")
    }

    // Pop returns pushed value
    got, ok := q.Pop()
    if !ok {
        t.Error("expected Pop() = true after Push()")
    }
    if got != val {
        t.Errorf("expected %v, got %v", val, got)
    }
}

func TestChannelQueue(t *testing.T) {
    q := queue.NewChannel[int](8)
    testQueue(t, q, 42)
}

func TestRingBuffer(t *testing.T) {
    q := queue.NewRingBuffer[int](8)
    testQueue(t, q, 42)
}

func TestChannelQueueFull(t *testing.T) {
    q := queue.NewChannel[int](2)
    q.Push(1)
    q.Push(2)
    if q.Push(3) {
        t.Error("expected Push() = false on full queue")
    }
}
```

### 3.2.1 SPSC Contract Violation Tests

These tests verify that the debug guards catch misuse:

```go
// queue_contract_test.go
package queue_test

import (
    "sync"
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

func TestRingBuffer_SPSC_ConcurrentPush_Panics(t *testing.T) {
    q := queue.NewRingBuffer[int](1024)

    defer func() {
        if r := recover(); r == nil {
            t.Error("expected panic on concurrent Push, but none occurred")
        }
    }()

    var wg sync.WaitGroup
    // Intentionally violate SPSC: multiple producers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            for j := 0; j < 1000; j++ {
                q.Push(n*1000 + j)
            }
        }(i)
    }
    wg.Wait()
}

func TestRingBuffer_SPSC_ConcurrentPop_Panics(t *testing.T) {
    q := queue.NewRingBuffer[int](1024)

    // Pre-fill
    for i := 0; i < 1024; i++ {
        q.Push(i)
    }

    defer func() {
        if r := recover(); r == nil {
            t.Error("expected panic on concurrent Pop, but none occurred")
        }
    }()

    var wg sync.WaitGroup
    // Intentionally violate SPSC: multiple consumers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                q.Pop()
            }
        }()
    }
    wg.Wait()
}
```

> **Note:** These tests are expected to panic. Run with `-tags=debug` to enable guards. In release mode (default), the guards may be compiled out for performance.

### 3.3 `internal/tick/tick_test.go`

```go
package tick_test

import (
    "testing"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func testTicker(t *testing.T, ticker tick.Ticker, interval time.Duration) {
    defer ticker.Stop()

    // Should not tick immediately
    if ticker.Tick() {
        t.Error("expected Tick() = false immediately")
    }

    // Wait for interval + buffer
    time.Sleep(interval + 10*time.Millisecond)

    // Should tick now
    if !ticker.Tick() {
        t.Error("expected Tick() = true after interval")
    }

    // Should not tick again immediately
    if ticker.Tick() {
        t.Error("expected Tick() = false immediately after tick")
    }
}

func TestStdTicker(t *testing.T) {
    testTicker(t, tick.NewTicker(50*time.Millisecond), 50*time.Millisecond)
}

func TestAtomicTicker(t *testing.T) {
    testTicker(t, tick.NewAtomicTicker(50*time.Millisecond), 50*time.Millisecond)
}

func TestBatchTicker(t *testing.T) {
    b := tick.NewBatch(50*time.Millisecond, 10)
    defer b.Stop()

    // First 9 calls should not tick (regardless of time)
    for i := 0; i < 9; i++ {
        if b.Tick() {
            t.Errorf("expected Tick() = false on call %d", i+1)
        }
    }

    // 10th call checks time - too soon
    if b.Tick() {
        t.Error("expected Tick() = false before interval")
    }

    // Wait and try again
    time.Sleep(60 * time.Millisecond)
    for i := 0; i < 10; i++ {
        b.Tick()
    }
    // The 10th should have triggered
}
```

### Exit Criteria
- [ ] `go test ./internal/...` passes
- [ ] Coverage > 80%: `go test -cover ./internal/...`

---

## Phase 4: Benchmark Tests

Each package gets a `_bench_test.go` file comparing implementations.

### Benchmark Conventions

- Use `b.ReportAllocs()` to track allocations
- Use `b.RunParallel()` for concurrency benchmarks
- Reset timer after setup: `b.ResetTimer()`
- Name format: `Benchmark<Package>_<Impl>_<Operation>`
- **Prevent compiler optimizations**: Use a package-level sink variable

### Preventing Dead Code Elimination

The compiler may optimize away loops where results are unused. Always sink results to a package-level variable:

```go
var sink bool // Package-level sink to prevent compiler optimization

func BenchmarkCancel_Atomic_Done(b *testing.B) {
    c := cancel.NewAtomic()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = c.Done()
    }
    sink = result // Sink prevents loop elimination
}
```

### Direct vs Interface Benchmarks

Interface method calls incur ~2-5ns overhead from dynamic dispatch. Include both:

```go
// Via interface (realistic usage)
func BenchmarkCancel_Atomic_Done_Interface(b *testing.B) {
    var c cancel.Canceler = cancel.NewAtomic()
    // ...
}

// Direct call (true floor)
func BenchmarkCancel_Atomic_Done_Direct(b *testing.B) {
    c := cancel.NewAtomic() // Concrete type
    // ...
}
```

### 4.1 `internal/cancel/cancel_bench_test.go`

```go
package cancel_test

import (
    "context"
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

var sinkBool bool // Prevent compiler from eliminating benchmark loops

// Direct type benchmarks (true performance floor)

func BenchmarkCancel_Context_Done_Direct(b *testing.B) {
    c := cancel.NewContext(context.Background())
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = c.Done()
    }
    sinkBool = result
}

func BenchmarkCancel_Atomic_Done_Direct(b *testing.B) {
    c := cancel.NewAtomic()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = c.Done()
    }
    sinkBool = result
}

// Interface benchmarks (realistic usage with dynamic dispatch)

func BenchmarkCancel_Context_Done_Interface(b *testing.B) {
    var c cancel.Canceler = cancel.NewContext(context.Background())
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = c.Done()
    }
    sinkBool = result
}

func BenchmarkCancel_Atomic_Done_Interface(b *testing.B) {
    var c cancel.Canceler = cancel.NewAtomic()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = c.Done()
    }
    sinkBool = result
}

// Parallel benchmarks

func BenchmarkCancel_Context_Done_Parallel(b *testing.B) {
    c := cancel.NewContext(context.Background())
    b.ReportAllocs()
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        var result bool
        for pb.Next() {
            result = c.Done()
        }
        sinkBool = result
    })
}

func BenchmarkCancel_Atomic_Done_Parallel(b *testing.B) {
    c := cancel.NewAtomic()
    b.ReportAllocs()
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        var result bool
        for pb.Next() {
            result = c.Done()
        }
        sinkBool = result
    })
}
```

### 4.2 `internal/queue/queue_bench_test.go`

```go
package queue_test

import (
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

var sinkInt int
var sinkOK bool

func BenchmarkQueue_Channel_PushPop_Direct(b *testing.B) {
    q := queue.NewChannel[int](1024)
    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok bool
    for i := 0; i < b.N; i++ {
        q.Push(i)
        val, ok = q.Pop()
    }
    sinkInt = val
    sinkOK = ok
}

func BenchmarkQueue_RingBuffer_PushPop_Direct(b *testing.B) {
    q := queue.NewRingBuffer[int](1024)
    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok bool
    for i := 0; i < b.N; i++ {
        q.Push(i)
        val, ok = q.Pop()
    }
    sinkInt = val
    sinkOK = ok
}

func BenchmarkQueue_Channel_PushPop_Interface(b *testing.B) {
    var q queue.Queue[int] = queue.NewChannel[int](1024)
    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok bool
    for i := 0; i < b.N; i++ {
        q.Push(i)
        val, ok = q.Pop()
    }
    sinkInt = val
    sinkOK = ok
}

func BenchmarkQueue_RingBuffer_PushPop_Interface(b *testing.B) {
    var q queue.Queue[int] = queue.NewRingBuffer[int](1024)
    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok bool
    for i := 0; i < b.N; i++ {
        q.Push(i)
        val, ok = q.Pop()
    }
    sinkInt = val
    sinkOK = ok
}
```

### 4.3 `internal/tick/tick_bench_test.go`

```go
package tick_test

import (
    "testing"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

const benchInterval = time.Hour // Long interval so Tick() returns false

var sinkTick bool

// Direct type benchmarks (true performance floor)

func BenchmarkTick_Std_Direct(b *testing.B) {
    t := tick.NewTicker(benchInterval)
    defer t.Stop()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

func BenchmarkTick_Batch_Direct(b *testing.B) {
    t := tick.NewBatch(benchInterval, 1000)
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

func BenchmarkTick_Atomic_Direct(b *testing.B) {
    t := tick.NewAtomicTicker(benchInterval)
    defer t.Stop()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

func BenchmarkTick_Nanotime_Direct(b *testing.B) {
    t := tick.NewNanotime(benchInterval)
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

func BenchmarkTick_TSC_Direct(b *testing.B) {
    t := tick.NewTSCCalibrated(benchInterval)
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

// Interface benchmarks (with dynamic dispatch overhead)

func BenchmarkTick_Std_Interface(b *testing.B) {
    var t tick.Ticker = tick.NewTicker(benchInterval)
    defer t.Stop()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

func BenchmarkTick_Atomic_Interface(b *testing.B) {
    var t tick.Ticker = tick.NewAtomicTicker(benchInterval)
    defer t.Stop()
    b.ReportAllocs()
    b.ResetTimer()

    var result bool
    for i := 0; i < b.N; i++ {
        result = t.Tick()
    }
    sinkTick = result
}

// Reset benchmark

func BenchmarkTick_Atomic_Reset(b *testing.B) {
    t := tick.NewAtomicTicker(benchInterval)
    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        t.Reset()
    }
}
```

### 4.4 Combined Interaction Benchmarks

**The most credible guidance** comes from testing realistic combinations, not isolated micro-costs.

Create `internal/combined/combined_bench_test.go`:

```go
package combined_test

import (
    "context"
    "testing"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
    "github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

var sinkInt int
var sinkBool bool

const benchInterval = time.Hour

// Realistic hot loop: check cancel + check tick + process message
// This is the pattern these optimizations are designed for.

func BenchmarkCombined_Standard(b *testing.B) {
    ctx := cancel.NewContext(context.Background())
    ticker := tick.NewTicker(benchInterval)
    q := queue.NewChannel[int](1024)
    defer ticker.Stop()

    // Pre-fill queue
    for i := 0; i < 1024; i++ {
        q.Push(i)
    }

    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok, cancelled, ticked bool
    for i := 0; i < b.N; i++ {
        cancelled = ctx.Done()
        ticked = ticker.Tick()
        val, ok = q.Pop()
        q.Push(val) // Recycle
    }
    sinkInt = val
    sinkBool = ok || cancelled || ticked
}

func BenchmarkCombined_Optimized(b *testing.B) {
    ctx := cancel.NewAtomic()
    ticker := tick.NewAtomicTicker(benchInterval)
    q := queue.NewRingBuffer[int](1024)

    // Pre-fill queue
    for i := 0; i < 1024; i++ {
        q.Push(i)
    }

    b.ReportAllocs()
    b.ResetTimer()

    var val int
    var ok, cancelled, ticked bool
    for i := 0; i < b.N; i++ {
        cancelled = ctx.Done()
        ticked = ticker.Tick()
        val, ok = q.Pop()
        q.Push(val) // Recycle
    }
    sinkInt = val
    sinkBool = ok || cancelled || ticked
}

// Simpler variant: just cancel + tick (no queue)
func BenchmarkCombined_CancelTick_Standard(b *testing.B) {
    ctx := cancel.NewContext(context.Background())
    ticker := tick.NewTicker(benchInterval)
    defer ticker.Stop()
    b.ReportAllocs()
    b.ResetTimer()

    var cancelled, ticked bool
    for i := 0; i < b.N; i++ {
        cancelled = ctx.Done()
        ticked = ticker.Tick()
    }
    sinkBool = cancelled || ticked
}

func BenchmarkCombined_CancelTick_Optimized(b *testing.B) {
    ctx := cancel.NewAtomic()
    ticker := tick.NewAtomicTicker(benchInterval)
    b.ReportAllocs()
    b.ResetTimer()

    var cancelled, ticked bool
    for i := 0; i < b.N; i++ {
        cancelled = ctx.Done()
        ticked = ticker.Tick()
    }
    sinkBool = cancelled || ticked
}
```

> **Why this matters:** Isolated benchmarks often show 10-20x speedups, but real loops have multiple operations. The combined benchmark shows the *actual* end-to-end improvement you'll see in production.

### 4.5 Two-Goroutine SPSC Pipeline Benchmark

The **most representative** benchmark for real Go systems—a producer/consumer pipeline:

```go
// internal/combined/pipeline_bench_test.go
package combined_test

import (
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

func BenchmarkPipeline_Channel(b *testing.B) {
    q := queue.NewChannel[int](1024)
    done := make(chan struct{})

    // Consumer
    go func() {
        for {
            select {
            case <-done:
                return
            default:
                q.Pop()
            }
        }
    }()

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        for !q.Push(i) {
            // Spin until push succeeds
        }
    }

    b.StopTimer()
    close(done)
}

func BenchmarkPipeline_RingBuffer(b *testing.B) {
    q := queue.NewRingBuffer[int](1024)
    done := make(chan struct{})

    // Consumer (single goroutine - SPSC contract)
    go func() {
        for {
            select {
            case <-done:
                return
            default:
                q.Pop()
            }
        }
    }()

    b.ReportAllocs()
    b.ResetTimer()

    // Producer (single goroutine - SPSC contract)
    for i := 0; i < b.N; i++ {
        for !q.Push(i) {
            // Spin until push succeeds
        }
    }

    b.StopTimer()
    close(done)
}
```

### 4.6 Benchmark Methodology Validation

Before declaring results valid, perform these checks:

#### Variance Check

Run benchmarks multiple times and verify low variance:

```bash
# Run 10 iterations
go test -bench=BenchmarkCancel -count=10 ./internal/cancel > results.txt

# Check variance with benchstat
benchstat results.txt
```

**Acceptable variance:** < 5% for most benchmarks. If higher, investigate:
- Background processes
- CPU frequency scaling
- Thermal throttling

#### Environment Lockdown Checklist

Before running "official" benchmarks:

```bash
# 1. Set CPU governor to performance
sudo cpupower frequency-set -g performance

# 2. Disable turbo boost (for consistent results)
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# 3. Check for background load
top -bn1 | head -20

# 4. Pin to single core (optional, for lowest variance)
taskset -c 0 go test -bench=. ./internal/...
```

#### Dead Code Elimination Check

Verify the compiler isn't optimizing away benchmark loops:

```bash
# Compile with assembly output
go test -c -o bench.test ./internal/cancel
go tool objdump -s 'BenchmarkCancel_Atomic_Done' bench.test | head -50

# Look for actual atomic load instructions, not empty loops
```

### Exit Criteria
- [ ] `go test -bench=. ./internal/...` runs without errors
- [ ] Results show expected performance ordering
- [ ] Combined benchmarks show meaningful speedup (>2x)
- [ ] `-count=10` runs show < 5% variance
- [ ] Environment lockdown checklist documented
- [ ] Assembly inspection confirms no dead code elimination

---

## Phase 5: CLI Tools

Each `cmd/` directory gets a `main.go` that demonstrates the library.

### 5.1 `cmd/context/main.go`

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

func main() {
    iterations := flag.Int("n", 10_000_000, "number of iterations")
    flag.Parse()

    // Benchmark context-based cancellation
    ctx := cancel.NewContext(context.Background())
    start := time.Now()
    for i := 0; i < *iterations; i++ {
        _ = ctx.Done()
    }
    ctxDur := time.Since(start)

    // Benchmark atomic-based cancellation
    atomic := cancel.NewAtomic()
    start = time.Now()
    for i := 0; i < *iterations; i++ {
        _ = atomic.Done()
    }
    atomicDur := time.Since(start)

    fmt.Printf("Context:  %v (%v/op)\n", ctxDur, ctxDur/time.Duration(*iterations))
    fmt.Printf("Atomic:   %v (%v/op)\n", atomicDur, atomicDur/time.Duration(*iterations))
    fmt.Printf("Speedup:  %.2fx\n", float64(ctxDur)/float64(atomicDur))
}
```

### 5.2 `cmd/channel/main.go`

```go
package main

import (
    "flag"
    "fmt"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

func main() {
    iterations := flag.Int("n", 10_000_000, "number of iterations")
    size := flag.Int("size", 1024, "queue size")
    flag.Parse()

    // Benchmark channel queue
    ch := queue.NewChannel[int](*size)
    start := time.Now()
    for i := 0; i < *iterations; i++ {
        ch.Push(i)
        ch.Pop()
    }
    chDur := time.Since(start)

    // Benchmark ring buffer
    ring := queue.NewRingBuffer[int](*size)
    start = time.Now()
    for i := 0; i < *iterations; i++ {
        ring.Push(i)
        ring.Pop()
    }
    ringDur := time.Since(start)

    fmt.Printf("Channel:  %v (%v/op)\n", chDur, chDur/time.Duration(*iterations))
    fmt.Printf("RingBuf:  %v (%v/op)\n", ringDur, ringDur/time.Duration(*iterations))
    fmt.Printf("Speedup:  %.2fx\n", float64(chDur)/float64(ringDur))
}
```

### 5.3 `cmd/ticker/main.go`

```go
package main

import (
    "flag"
    "fmt"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func main() {
    iterations := flag.Int("n", 10_000_000, "number of iterations")
    flag.Parse()

    interval := time.Hour // Long so we measure check overhead, not actual ticks

    impls := []struct {
        name   string
        ticker tick.Ticker
    }{
        {"StdTicker", tick.NewTicker(interval)},
        {"Batch", tick.NewBatch(interval, 1000)},
        {"Atomic", tick.NewAtomicTicker(interval)},
        {"Nanotime", tick.NewNanotime(interval)},
        {"TSC", tick.NewTSC(interval, 3.0)},
    }

    results := make([]time.Duration, len(impls))

    for i, impl := range impls {
        start := time.Now()
        for j := 0; j < *iterations; j++ {
            _ = impl.ticker.Tick()
        }
        results[i] = time.Since(start)
        impl.ticker.Stop()
    }

    fmt.Printf("\nResults (%d iterations):\n", *iterations)
    fmt.Println("─────────────────────────────────────")
    baseline := results[0]
    for i, impl := range impls {
        fmt.Printf("%-12s %12v  %6.2fx\n",
            impl.name,
            results[i],
            float64(baseline)/float64(results[i]))
    }
}
```

### 5.4 `cmd/context-ticker/main.go`

Combined benchmark showing cumulative overhead of checking both context and ticker.

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "time"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
    "github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func main() {
    iterations := flag.Int("n", 10_000_000, "number of iterations")
    flag.Parse()

    interval := time.Hour

    // Standard: context + ticker via select
    ctxCancel := cancel.NewContext(context.Background())
    stdTicker := tick.NewTicker(interval)
    start := time.Now()
    for i := 0; i < *iterations; i++ {
        _ = ctxCancel.Done()
        _ = stdTicker.Tick()
    }
    stdDur := time.Since(start)
    stdTicker.Stop()

    // Optimized: atomic cancel + nanotime ticker
    atomicCancel := cancel.NewAtomic()
    nanoTicker := tick.NewNanotime(interval)
    start = time.Now()
    for i := 0; i < *iterations; i++ {
        _ = atomicCancel.Done()
        _ = nanoTicker.Tick()
    }
    optDur := time.Since(start)

    fmt.Printf("Standard (ctx+ticker):    %v\n", stdDur)
    fmt.Printf("Optimized (atomic+nano):  %v\n", optDur)
    fmt.Printf("Speedup:                  %.2fx\n", float64(stdDur)/float64(optDur))
}
```

### Exit Criteria
- [ ] `go build ./cmd/...` succeeds
- [ ] All binaries run and produce output
- [ ] `go run ./cmd/context -n 1000000` completes in reasonable time

---

## Phase 6: Validation

### 6.1 Race Detection

Run all tests with the race detector:

```bash
# Unit tests with race detection
go test -race ./internal/...

# Benchmarks with race detection (slower, but catches issues)
go test -race -bench=. -benchtime=100ms ./internal/...
```

**Focus areas for race conditions:**
- `AtomicCanceler`: concurrent `Done()` and `Cancel()` calls
- `AtomicTicker`: concurrent `Tick()` calls with CAS
- `RingBuffer`: SPSC contract (single producer, single consumer)

### 6.2 Add Race-Specific Tests

```go
// internal/cancel/cancel_race_test.go
package cancel_test

import (
    "sync"
    "testing"

    "github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

func TestAtomicCanceler_Race(t *testing.T) {
    c := cancel.NewAtomic()
    var wg sync.WaitGroup

    // Spawn readers
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < 10000; j++ {
                _ = c.Done()
            }
        }()
    }

    // Spawn writer
    wg.Add(1)
    go func() {
        defer wg.Done()
        c.Cancel()
    }()

    wg.Wait()

    if !c.Done() {
        t.Error("expected Done() = true after Cancel()")
    }
}
```

### 6.3 CPU Profiling

```bash
# Profile a benchmark
go test -bench=BenchmarkCancel_Context_Done -cpuprofile=cpu.prof ./internal/cancel
go tool pprof -http=:8080 cpu.prof

# Profile a CLI tool
go run ./cmd/ticker -n 100000000 &
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
```

### 6.4 Memory Profiling

```bash
# Check for allocations
go test -bench=. -benchmem ./internal/...

# Expected: optimized implementations should show 0 allocs/op
```

### 6.5 Documentation: Debug vs Release Modes

Document clearly in the README and package docs:

```go
// Package queue provides SPSC queue implementations for benchmarking.
//
// # RingBuffer Safety
//
// RingBuffer is a Single-Producer Single-Consumer (SPSC) queue.
// It is NOT safe for multiple goroutines to call Push() or Pop() concurrently.
//
// Build with -tags=debug to enable runtime guards that panic on misuse:
//
//     go test -tags=debug ./internal/queue
//
// In release mode (default), guards are disabled for maximum performance.
// Misuse in release mode results in undefined behavior (data races, corruption).
package queue
```

### 6.6 Environment Documentation

Create `BENCHMARKING.md` with reproducibility instructions:

```markdown
# Benchmarking Environment

## Hardware Used
- CPU: [your CPU model]
- Cores: [count]
- RAM: [size]
- OS: [version]
- Go: [version]

## Environment Setup

### Linux (recommended)

# Set performance governor
sudo cpupower frequency-set -g performance

# Disable turbo boost
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# Check CPU frequency is stable
watch -n1 "cat /proc/cpuinfo | grep MHz"

### Running Benchmarks

# Full benchmark suite with 10 iterations
go test -bench=. -count=10 -benchmem ./internal/... | tee results.txt

# Analyze with benchstat
benchstat results.txt

## Known Variance Sources
- Background processes (close browsers, IDEs)
- Thermal throttling (let CPU cool between runs)
- NUMA effects (pin to single socket if applicable)
```

### Exit Criteria
- [ ] `go test -race ./...` passes
- [ ] No unexpected allocations in hot paths
- [ ] Profiling confirms expected performance characteristics
- [ ] Debug mode documented with `-tags=debug`
- [ ] Release mode warnings documented
- [ ] `BENCHMARKING.md` created with environment notes

---

## Summary Checklist

| Phase | Task | Status |
|-------|------|--------|
| 1 | `go.mod` created | ☐ |
| 1 | Directory structure created | ☐ |
| 1 | Makefile created | ☐ |
| 2 | `internal/cancel` implemented | ☐ |
| 2 | `internal/queue` implemented | ☐ |
| 2 | `internal/tick` implemented | ☐ |
| 2 | Consolidate AtomicTicker/NanotimeTicker | ☐ |
| 2.5 | Build tags for safe defaults | ☐ |
| 2.5 | TSC stub for non-amd64 | ☐ |
| 2.5 | CI matrix (amd64/arm64, Go versions) | ☐ |
| 2.5 | `-tags=go_safe` fallback works | ☐ |
| 3 | Unit tests for `cancel` | ☐ |
| 3 | Unit tests for `queue` | ☐ |
| 3 | Unit tests for `tick` | ☐ |
| 3 | SPSC violation tests (panic in debug mode) | ☐ |
| 4 | Benchmarks for `cancel` | ☐ |
| 4 | Benchmarks for `queue` | ☐ |
| 4 | Benchmarks for `tick` | ☐ |
| 4 | Combined interaction benchmarks | ☐ |
| 4 | SPSC pipeline benchmark (2 goroutines) | ☐ |
| 4 | Variance check (`-count=10`, < 5%) | ☐ |
| 4 | Dead code elimination verified | ☐ |
| 5 | `cmd/context` | ☐ |
| 5 | `cmd/channel` | ☐ |
| 5 | `cmd/ticker` | ☐ |
| 5 | `cmd/context-ticker` | ☐ |
| 6 | Race detection passes | ☐ |
| 6 | Profiling complete | ☐ |
| 6 | Debug/release modes documented | ☐ |
| 6 | `BENCHMARKING.md` created | ☐ |

---

## Appendix: Expected Benchmark Results

Based on typical measurements, expect roughly:

| Operation | Standard | Optimized | Speedup |
|-----------|----------|-----------|---------|
| `ctx.Done()` check | ~15-25ns | ~1-2ns | 10-20x |
| Channel push/pop | ~50-100ns | ~10-20ns | 3-5x |
| Ticker check | ~20-40ns | ~2-5ns | 5-10x |
| Combined (ctx+tick) | ~50-80ns | ~5-10ns | 8-15x |

*Actual results vary by CPU, Go version, and system load.*
