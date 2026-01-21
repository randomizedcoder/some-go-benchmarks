//go:build amd64

package tick_test

import (
	"testing"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func TestTSCTicker(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewTSCCalibrated(interval)
	defer ticker.Stop()

	// Should not tick immediately
	if ticker.Tick() {
		t.Error("expected Tick() = false immediately after creation")
	}

	// Wait for interval + buffer
	time.Sleep(interval + 20*time.Millisecond)

	// Should tick now
	if !ticker.Tick() {
		t.Error("expected Tick() = true after interval elapsed")
	}

	// Should not tick again immediately
	if ticker.Tick() {
		t.Error("expected Tick() = false immediately after tick")
	}
}

func TestTSCTicker_Reset(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewTSCCalibrated(interval)
	defer ticker.Stop()

	// Wait and tick
	time.Sleep(interval + 20*time.Millisecond)
	if !ticker.Tick() {
		t.Error("expected Tick() = true after interval")
	}

	// Reset
	ticker.Reset()

	// Should not tick immediately after reset
	if ticker.Tick() {
		t.Error("expected Tick() = false after Reset()")
	}
}

func TestCalibrateTSC(t *testing.T) {
	cyclesPerNs := tick.CalibrateTSC()

	// Sanity check: should be between 0.5 and 10 cycles/ns
	// (500MHz to 10GHz CPUs)
	if cyclesPerNs < 0.5 || cyclesPerNs > 10 {
		t.Errorf("CalibrateTSC() = %f, expected between 0.5 and 10", cyclesPerNs)
	}

	t.Logf("Calibrated TSC: %.2f cycles/ns (%.2f GHz equivalent)", cyclesPerNs, cyclesPerNs)
}

func TestTSCTicker_CyclesPerNs(t *testing.T) {
	ticker := tick.NewTSC(time.Second, 3.0)
	if ticker.CyclesPerNs() != 3.0 {
		t.Errorf("expected CyclesPerNs() = 3.0, got %f", ticker.CyclesPerNs())
	}
}
