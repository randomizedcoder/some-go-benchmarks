package tick

import "time"

// BatchTicker checks the time only every N calls to Tick().
//
// This reduces the overhead of time checks by amortizing them across
// multiple loop iterations. Useful when processing items rapidly and
// you don't need sub-millisecond precision on tick timing.
//
// Example: With every=1000 and interval=100ms, the time is checked
// only once per 1000 calls, and a tick fires if 100ms has passed.
type BatchTicker struct {
	interval time.Duration
	every    int
	count    int
	lastTick time.Time
}

// NewBatch creates a BatchTicker that checks time every N operations.
//
// Parameters:
//   - interval: How often ticks should fire (wall clock time)
//   - every: Check the clock only every N calls to Tick()
func NewBatch(interval time.Duration, every int) *BatchTicker {
	if every < 1 {
		every = 1
	}
	return &BatchTicker{
		interval: interval,
		every:    every,
		lastTick: time.Now(),
	}
}

// Tick returns true if the interval has elapsed.
//
// The time is only checked every N calls (as specified by 'every').
// On other calls, this returns false immediately without checking time.
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

// Reset resets the ticker state.
func (b *BatchTicker) Reset() {
	b.count = 0
	b.lastTick = time.Now()
}

// Stop is a no-op for BatchTicker (no resources to release).
func (b *BatchTicker) Stop() {}

// Every returns the batch size.
func (b *BatchTicker) Every() int {
	return b.every
}

// Interval returns the ticker's interval.
func (b *BatchTicker) Interval() time.Duration {
	return b.interval
}
