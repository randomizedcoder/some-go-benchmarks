// Command context benchmarks context cancellation checking.
//
// Usage:
//
//	go run ./cmd/context -n 10000000
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

	fmt.Printf("Benchmarking cancellation check (%d iterations)\n", *iterations)
	fmt.Println("─────────────────────────────────────────────────")

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

	// Results
	ctxPerOp := float64(ctxDur.Nanoseconds()) / float64(*iterations)
	atomicPerOp := float64(atomicDur.Nanoseconds()) / float64(*iterations)

	fmt.Printf("\nResults:\n")
	fmt.Printf("  Context:  %v (%.2f ns/op)\n", ctxDur, ctxPerOp)
	fmt.Printf("  Atomic:   %v (%.2f ns/op)\n", atomicDur, atomicPerOp)
	fmt.Printf("\n  Speedup:  %.2fx\n", ctxPerOp/atomicPerOp)

	// Extrapolate to ops/second
	fmt.Printf("\nThroughput (theoretical max):\n")
	fmt.Printf("  Context:  %.2f M ops/sec\n", 1000/ctxPerOp)
	fmt.Printf("  Atomic:   %.2f M ops/sec\n", 1000/atomicPerOp)
}
