// Command channel benchmarks SPSC queue implementations.
//
// Usage:
//
//	go run ./cmd/channel -n 10000000 -size 1024
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

	fmt.Printf("Benchmarking SPSC queue (%d iterations, size=%d)\n", *iterations, *size)
	fmt.Println("─────────────────────────────────────────────────")

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

	// Results
	chPerOp := float64(chDur.Nanoseconds()) / float64(*iterations)
	ringPerOp := float64(ringDur.Nanoseconds()) / float64(*iterations)

	fmt.Printf("\nResults (push + pop per iteration):\n")
	fmt.Printf("  Channel:     %v (%.2f ns/op)\n", chDur, chPerOp)
	fmt.Printf("  RingBuffer:  %v (%.2f ns/op)\n", ringDur, ringPerOp)

	if ringPerOp < chPerOp {
		fmt.Printf("\n  Speedup:  %.2fx (RingBuffer faster)\n", chPerOp/ringPerOp)
	} else {
		fmt.Printf("\n  Speedup:  %.2fx (Channel faster)\n", ringPerOp/chPerOp)
	}

	// Extrapolate to ops/second
	fmt.Printf("\nThroughput (theoretical max):\n")
	fmt.Printf("  Channel:     %.2f M ops/sec\n", 1000/chPerOp)
	fmt.Printf("  RingBuffer:  %.2f M ops/sec\n", 1000/ringPerOp)
}
