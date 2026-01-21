//go:build amd64

package tick

import (
	"sync/atomic"
	"time"
)

// rdtsc reads the CPU's Time Stamp Counter.
// Implemented in tsc_amd64.s
func rdtsc() uint64

// CalibrateTSC measures CPU cycles per nanosecond.
//
// This performs a ~10ms calibration by comparing TSC ticks against
// wall clock time. The result is approximate and can vary with:
//   - CPU frequency scaling (Turbo Boost, SpeedStep)
//   - Power management states
//   - Thermal throttling
//
// For best results, run on a warmed-up CPU with frequency governor
// set to "performance".
func CalibrateTSC() float64 {
	// Warm up the TSC path
	rdtsc()
	rdtsc()

	start := rdtsc()
	t1 := time.Now()
	time.Sleep(10 * time.Millisecond)
	end := rdtsc()
	t2 := time.Now()

	cycles := float64(end - start)
	nanos := float64(t2.Sub(t1).Nanoseconds())

	return cycles / nanos
}

// TSCTicker uses the CPU's Time Stamp Counter for ultra-low-latency tick checks.
//
// This is the fastest possible ticker on x86, bypassing the OS entirely.
// However, it requires calibration and may drift with CPU frequency changes.
//
// Typical performance:
//   - AtomicTicker.Tick(): ~3-5ns
//   - TSCTicker.Tick(): ~1-2ns
//
// Use NewTSCCalibrated for automatic calibration, or NewTSC if you've
// pre-measured your CPU's cycles-per-nanosecond ratio.
type TSCTicker struct {
	intervalCycles uint64
	lastTick       atomic.Uint64
	cyclesPerNs    float64
}

// NewTSC creates a TSCTicker with an explicit cycles-per-nanosecond ratio.
//
// Parameters:
//   - interval: The tick interval
//   - cyclesPerNs: CPU cycles per nanosecond (e.g., 3.0 for a 3GHz CPU)
func NewTSC(interval time.Duration, cyclesPerNs float64) *TSCTicker {
	t := &TSCTicker{
		intervalCycles: uint64(float64(interval.Nanoseconds()) * cyclesPerNs),
		cyclesPerNs:    cyclesPerNs,
	}
	t.lastTick.Store(rdtsc())
	return t
}

// NewTSCCalibrated creates a TSCTicker with automatic calibration.
//
// This blocks for ~10ms while calibrating. For production use,
// consider calibrating once at startup and reusing the ratio.
func NewTSCCalibrated(interval time.Duration) *TSCTicker {
	return NewTSC(interval, CalibrateTSC())
}

// Tick returns true if the interval has elapsed since the last tick.
func (t *TSCTicker) Tick() bool {
	now := rdtsc()
	last := t.lastTick.Load()

	if now-last >= t.intervalCycles {
		if t.lastTick.CompareAndSwap(last, now) {
			return true
		}
	}
	return false
}

// Reset resets the ticker to start a new interval from now.
func (t *TSCTicker) Reset() {
	t.lastTick.Store(rdtsc())
}

// Stop is a no-op for TSCTicker (no resources to release).
func (t *TSCTicker) Stop() {}

// CyclesPerNs returns the calibrated cycles-per-nanosecond ratio.
func (t *TSCTicker) CyclesPerNs() float64 {
	return t.cyclesPerNs
}
