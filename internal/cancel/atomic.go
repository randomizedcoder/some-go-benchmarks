package cancel

import "sync/atomic"

// AtomicCanceler uses an atomic.Bool for cancellation signaling.
//
// This is the optimized approach. Each call to Done() performs
// a single atomic load, which is much faster than a channel select.
//
// Typical performance:
//   - ContextCanceler.Done(): ~15-25ns
//   - AtomicCanceler.Done(): ~1-2ns
type AtomicCanceler struct {
	done atomic.Bool
}

// NewAtomic creates a new AtomicCanceler.
func NewAtomic() *AtomicCanceler {
	return &AtomicCanceler{}
}

// Done returns true if cancellation has been triggered.
//
// This performs a single atomic load operation.
func (a *AtomicCanceler) Done() bool {
	return a.done.Load()
}

// Cancel triggers cancellation.
//
// Safe to call multiple times; subsequent calls are no-ops.
func (a *AtomicCanceler) Cancel() {
	a.done.Store(true)
}

// Reset clears the cancellation flag.
//
// Useful for reusing the canceler without reallocation.
// Not safe to call concurrently with Done() or Cancel().
func (a *AtomicCanceler) Reset() {
	a.done.Store(false)
}
