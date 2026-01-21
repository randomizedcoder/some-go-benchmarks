// Package cancel provides cancellation signaling implementations for benchmarking.
//
// This package offers two implementations of the Canceler interface:
//   - ContextCanceler: Standard library approach using context.Context
//   - AtomicCanceler: Optimized approach using atomic.Bool
//
// The atomic approach is significantly faster in polling hot-loops where
// Done() is called millions of times per second.
package cancel

// Canceler provides cancellation signaling to workers.
//
// Implementations must be safe for concurrent use:
//   - Multiple goroutines may call Done() concurrently
//   - Cancel() may be called concurrently with Done()
type Canceler interface {
	// Done returns true if cancellation has been triggered.
	Done() bool

	// Cancel triggers cancellation. Safe to call multiple times.
	Cancel()
}
