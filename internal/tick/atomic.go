package tick

import (
	"sync/atomic"
	"time"
	_ "unsafe" // Required for go:linkname
)

// nanotime returns the current monotonic time in nanoseconds.
// This is faster than time.Now() because it returns a single int64
// and avoids constructing a time.Time struct.
//
// Note: This uses go:linkname to access an internal runtime function.
// It may break in future Go versions, though it has been stable.
//
//go:linkname nanotime runtime.nanotime
func nanotime() int64

// AtomicTicker uses atomic operations and runtime.nanotime for fast tick checks.
//
// This is the recommended optimized ticker for most use cases.
// It uses the runtime's internal monotonic clock (faster than time.Now())
// and atomic operations for thread-safe tick detection.
//
// Typical performance:
//   - StdTicker.Tick(): ~20-40ns
//   - AtomicTicker.Tick(): ~3-5ns
type AtomicTicker struct {
	interval int64 // nanoseconds
	lastTick atomic.Int64
}

// NewAtomicTicker creates an AtomicTicker with the specified interval.
func NewAtomicTicker(interval time.Duration) *AtomicTicker {
	t := &AtomicTicker{
		interval: int64(interval),
	}
	t.lastTick.Store(nanotime())
	return t
}

// Tick returns true if the interval has elapsed since the last tick.
//
// Uses a compare-and-swap to prevent multiple goroutines from
// triggering the same tick (though typically only one goroutine polls).
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

// Reset resets the ticker to start a new interval from now.
func (a *AtomicTicker) Reset() {
	a.lastTick.Store(nanotime())
}

// Stop is a no-op for AtomicTicker (no resources to release).
func (a *AtomicTicker) Stop() {}

// Interval returns the ticker's interval.
func (a *AtomicTicker) Interval() time.Duration {
	return time.Duration(a.interval)
}
