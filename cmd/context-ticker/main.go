// Command context-ticker benchmarks combined cancellation + tick checking.
//
// This represents a realistic hot-loop pattern where you check both
// context cancellation and periodic timing on every iteration.
//
// Usage:
//
//	go run ./cmd/context-ticker -n 10000000
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

	interval := time.Hour // Long so we measure check overhead, not actual ticks

	fmt.Printf("Benchmarking combined cancel+tick check (%d iterations)\n", *iterations)
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("This simulates a hot loop that checks for cancellation")
	fmt.Println("and periodic timing on every iteration:")
	fmt.Println()
	fmt.Println("  for {")
	fmt.Println("      if cancel.Done() { return }")
	fmt.Println("      if ticker.Tick() { doPeriodicWork() }")
	fmt.Println("      processItem()")
	fmt.Println("  }")
	fmt.Println()

	// Standard: context + time.Ticker
	ctxCancel := cancel.NewContext(context.Background())
	stdTicker := tick.NewTicker(interval)

	start := time.Now()
	for i := 0; i < *iterations; i++ {
		_ = ctxCancel.Done()
		_ = stdTicker.Tick()
	}
	stdDur := time.Since(start)
	stdTicker.Stop()

	// Optimized: atomic cancel + atomic ticker
	atomicCancel := cancel.NewAtomic()
	atomicTicker := tick.NewAtomicTicker(interval)

	start = time.Now()
	for i := 0; i < *iterations; i++ {
		_ = atomicCancel.Done()
		_ = atomicTicker.Tick()
	}
	optDur := time.Since(start)

	// Ultra-optimized: atomic cancel + batch ticker
	atomicCancel2 := cancel.NewAtomic()
	batchTicker := tick.NewBatch(interval, 1000)

	start = time.Now()
	for i := 0; i < *iterations; i++ {
		_ = atomicCancel2.Done()
		_ = batchTicker.Tick()
	}
	batchDur := time.Since(start)

	// Results
	stdPerOp := float64(stdDur.Nanoseconds()) / float64(*iterations)
	optPerOp := float64(optDur.Nanoseconds()) / float64(*iterations)
	batchPerOp := float64(batchDur.Nanoseconds()) / float64(*iterations)

	fmt.Println("Results:")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Printf("  Standard (ctx + time.Ticker):\n")
	fmt.Printf("    Total: %v, Per-op: %.2f ns\n", stdDur, stdPerOp)
	fmt.Println()
	fmt.Printf("  Optimized (atomic + AtomicTicker):\n")
	fmt.Printf("    Total: %v, Per-op: %.2f ns\n", optDur, optPerOp)
	fmt.Printf("    Speedup: %.2fx\n", stdPerOp/optPerOp)
	fmt.Println()
	fmt.Printf("  Ultra (atomic + BatchTicker):\n")
	fmt.Printf("    Total: %v, Per-op: %.2f ns\n", batchDur, batchPerOp)
	fmt.Printf("    Speedup: %.2fx\n", stdPerOp/batchPerOp)
	fmt.Println()

	// Impact analysis
	fmt.Println("Impact Analysis:")
	fmt.Println("─────────────────────────────────────────────────────────")
	savedNs := stdPerOp - optPerOp

	fmt.Printf("  Savings per iteration: %.2f ns\n", savedNs)
	fmt.Println()

	rates := []int{100_000, 1_000_000, 10_000_000}
	for _, rate := range rates {
		savedPerSec := savedNs * float64(rate) / 1e9
		fmt.Printf("  At %dK ops/sec: save %.2f ms/sec (%.2f%% of 1 core)\n",
			rate/1000, savedPerSec*1000, savedPerSec*100)
	}
}
