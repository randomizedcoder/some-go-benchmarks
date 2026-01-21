package cancel_test

import (
	"context"
	"sync"
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

// TestContextCanceler_Race tests concurrent access to ContextCanceler.
// Run with: go test -race ./internal/cancel
func TestContextCanceler_Race(t *testing.T) {
	c := cancel.NewContext(context.Background())
	var wg sync.WaitGroup

	// Spawn readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				_ = c.Done()
			}
		}()
	}

	// Spawn writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Cancel()
	}()

	wg.Wait()

	if !c.Done() {
		t.Error("expected Done() = true after Cancel()")
	}
}

// TestAtomicCanceler_Race tests concurrent access to AtomicCanceler.
// Run with: go test -race ./internal/cancel
func TestAtomicCanceler_Race(t *testing.T) {
	c := cancel.NewAtomic()
	var wg sync.WaitGroup

	// Spawn readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				_ = c.Done()
			}
		}()
	}

	// Spawn writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Cancel()
	}()

	wg.Wait()

	if !c.Done() {
		t.Error("expected Done() = true after Cancel()")
	}
}
