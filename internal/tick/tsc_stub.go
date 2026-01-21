//go:build !amd64

package tick

import (
	"errors"
	"time"
)

// ErrTSCNotSupported is returned when TSC is not available on this architecture.
var ErrTSCNotSupported = errors.New("tick: TSC ticker requires amd64 architecture")

// TSCTicker is a stub for non-amd64 architectures.
// Use AtomicTicker instead for cross-platform code.
type TSCTicker struct{}

// CalibrateTSC returns an error on non-amd64 architectures.
func CalibrateTSC() (float64, error) {
	return 0, ErrTSCNotSupported
}

// NewTSC returns an error on non-amd64 architectures.
func NewTSC(interval time.Duration, cyclesPerNs float64) (*TSCTicker, error) {
	return nil, ErrTSCNotSupported
}

// NewTSCCalibrated returns an error on non-amd64 architectures.
func NewTSCCalibrated(interval time.Duration) (*TSCTicker, error) {
	return nil, ErrTSCNotSupported
}

// Tick always returns false on stub implementation.
func (t *TSCTicker) Tick() bool { return false }

// Reset is a no-op on stub implementation.
func (t *TSCTicker) Reset() {}

// Stop is a no-op on stub implementation.
func (t *TSCTicker) Stop() {}

// CyclesPerNs returns 0 on stub implementation.
func (t *TSCTicker) CyclesPerNs() float64 { return 0 }
