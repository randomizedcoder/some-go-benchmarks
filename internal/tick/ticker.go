package tick

import "time"

// StdTicker wraps time.Ticker for the Ticker interface.
//
// This is the standard library approach. Each call to Tick() performs
// a non-blocking select on the ticker's channel.
type StdTicker struct {
	ticker   *time.Ticker
	interval time.Duration
}

// NewTicker creates a StdTicker with the specified interval.
func NewTicker(interval time.Duration) *StdTicker {
	return &StdTicker{
		ticker:   time.NewTicker(interval),
		interval: interval,
	}
}

// Tick returns true if the interval has elapsed.
// This performs a non-blocking select on the ticker channel.
func (t *StdTicker) Tick() bool {
	select {
	case <-t.ticker.C:
		return true
	default:
		return false
	}
}

// Reset resets the ticker to start a new interval from now.
func (t *StdTicker) Reset() {
	t.ticker.Reset(t.interval)
}

// Stop stops the ticker and releases resources.
func (t *StdTicker) Stop() {
	t.ticker.Stop()
}

// Interval returns the ticker's interval.
func (t *StdTicker) Interval() time.Duration {
	return t.interval
}
