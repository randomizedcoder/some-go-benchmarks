// Package tick provides periodic trigger implementations for benchmarking.
//
// This package offers several implementations of the Ticker interface:
//   - StdTicker: Standard library time.Ticker wrapper
//   - BatchTicker: Check only every N operations
//   - AtomicTicker: Atomic timestamp comparison using runtime.nanotime
//   - TSCTicker: Raw CPU timestamp counter (x86 only)
//
// The optimized implementations avoid the overhead of the Go runtime's
// central timer heap, which can be significant in high-throughput loops.
package tick

import "time"

// Ticker signals when a time interval has elapsed.
//
// All implementations are safe for concurrent use from multiple goroutines,
// though typically only one goroutine polls Tick() in a hot loop.
type Ticker interface {
	// Tick returns true if the interval has elapsed since the last tick.
	// This is a non-blocking check.
	Tick() bool

	// Reset resets the ticker to start a new interval from now.
	// Useful for reusing a ticker without reallocation.
	Reset()

	// Stop releases any resources held by the ticker.
	// After Stop, the ticker should not be used.
	Stop()
}

// DefaultInterval is a reasonable default for testing.
const DefaultInterval = 100 * time.Millisecond
