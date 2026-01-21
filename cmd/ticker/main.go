// Command ticker benchmarks periodic tick checking implementations.
//
// Usage:
//
//	go run ./cmd/ticker -n 10000000
package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

type tickerInfo struct {
	name   string
	create func() tick.Ticker
}

func main() {
	iterations := flag.Int("n", 10_000_000, "number of iterations")
	flag.Parse()

	interval := time.Hour // Long so we measure check overhead, not actual ticks

	fmt.Printf("Benchmarking tick check (%d iterations)\n", *iterations)
	fmt.Printf("Architecture: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println("─────────────────────────────────────────────────")

	// Build list of tickers to test
	tickers := []tickerInfo{
		{"StdTicker", func() tick.Ticker { return tick.NewTicker(interval) }},
		{"BatchTicker(1000)", func() tick.Ticker { return tick.NewBatch(interval, 1000) }},
		{"AtomicTicker", func() tick.Ticker { return tick.NewAtomicTicker(interval) }},
	}

	// Add TSC ticker only on amd64
	if runtime.GOARCH == "amd64" {
		tickers = append(tickers, tickerInfo{
			"TSCTicker",
			func() tick.Ticker { return tick.NewTSCCalibrated(interval) },
		})
	}

	results := make([]time.Duration, len(tickers))

	for i, info := range tickers {
		t := info.create()
		start := time.Now()
		for j := 0; j < *iterations; j++ {
			_ = t.Tick()
		}
		results[i] = time.Since(start)
		t.Stop()
	}

	// Print results
	fmt.Printf("\nResults:\n")
	baseline := float64(results[0].Nanoseconds()) / float64(*iterations)

	for i, info := range tickers {
		perOp := float64(results[i].Nanoseconds()) / float64(*iterations)
		speedup := baseline / perOp
		throughput := 1000 / perOp // M ops/sec

		fmt.Printf("  %-20s %12v  %8.2f ns/op  %6.2fx  %8.2f M/s\n",
			info.name, results[i], perOp, speedup, throughput)
	}

	fmt.Printf("\nNote: BatchTicker only checks time every N calls, so overhead is amortized.\n")
}
