package tick_test

import (
	"testing"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func TestStdTicker(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewTicker(interval)
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

func TestStdTicker_Reset(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewTicker(interval)
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

func TestAtomicTicker(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewAtomicTicker(interval)
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

func TestAtomicTicker_Reset(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewAtomicTicker(interval)
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

func TestBatchTicker(t *testing.T) {
	interval := 50 * time.Millisecond
	every := 10
	ticker := tick.NewBatch(interval, every)
	defer ticker.Stop()

	// First 9 calls should not tick (regardless of time)
	for i := 0; i < every-1; i++ {
		if ticker.Tick() {
			t.Errorf("expected Tick() = false on call %d (before batch)", i+1)
		}
	}

	// 10th call checks time - but interval hasn't passed
	if ticker.Tick() {
		t.Error("expected Tick() = false before interval elapsed")
	}

	// Wait for interval
	time.Sleep(interval + 20*time.Millisecond)

	// Now do another batch
	for i := 0; i < every-1; i++ {
		ticker.Tick() // These don't check time
	}

	// The Nth call should tick
	if !ticker.Tick() {
		t.Error("expected Tick() = true after interval elapsed and batch complete")
	}
}

func TestBatchTicker_Reset(t *testing.T) {
	interval := 50 * time.Millisecond
	ticker := tick.NewBatch(interval, 10)
	defer ticker.Stop()

	// Call a few times
	for i := 0; i < 5; i++ {
		ticker.Tick()
	}

	// Reset
	ticker.Reset()

	// Should be back to initial state
	// Call 9 times (none should tick)
	for i := 0; i < 9; i++ {
		if ticker.Tick() {
			t.Errorf("expected Tick() = false on call %d after Reset()", i+1)
		}
	}
}

func TestBatchTicker_Every(t *testing.T) {
	ticker := tick.NewBatch(time.Second, 100)
	if ticker.Every() != 100 {
		t.Errorf("expected Every() = 100, got %d", ticker.Every())
	}
}

// Test that all implementations satisfy the interface
func TestTickerInterface(t *testing.T) {
	interval := 50 * time.Millisecond

	// Factory functions to create fresh tickers for each test
	testCases := []struct {
		name   string
		create func() tick.Ticker
	}{
		{"StdTicker", func() tick.Ticker { return tick.NewTicker(interval) }},
		{"AtomicTicker", func() tick.Ticker { return tick.NewAtomicTicker(interval) }},
		{"BatchTicker", func() tick.Ticker { return tick.NewBatch(interval, 1) }}, // every=1 so it checks time on every call
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a fresh ticker for this subtest
			ticker := tc.create()
			defer ticker.Stop()

			// Should not tick immediately
			if ticker.Tick() {
				t.Error("expected Tick() = false immediately")
			}

			// Wait and check
			time.Sleep(interval + 20*time.Millisecond)

			if !ticker.Tick() {
				t.Error("expected Tick() = true after interval")
			}
		})
	}
}
