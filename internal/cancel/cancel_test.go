package cancel_test

import (
	"context"
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

func TestContextCanceler(t *testing.T) {
	c := cancel.NewContext(context.Background())

	if c.Done() {
		t.Error("expected Done() = false before Cancel()")
	}

	c.Cancel()

	if !c.Done() {
		t.Error("expected Done() = true after Cancel()")
	}

	// Verify idempotent
	c.Cancel()
	if !c.Done() {
		t.Error("expected Done() = true after second Cancel()")
	}
}

func TestAtomicCanceler(t *testing.T) {
	c := cancel.NewAtomic()

	if c.Done() {
		t.Error("expected Done() = false before Cancel()")
	}

	c.Cancel()

	if !c.Done() {
		t.Error("expected Done() = true after Cancel()")
	}

	// Verify idempotent
	c.Cancel()
	if !c.Done() {
		t.Error("expected Done() = true after second Cancel()")
	}
}

func TestAtomicCanceler_Reset(t *testing.T) {
	c := cancel.NewAtomic()

	c.Cancel()
	if !c.Done() {
		t.Error("expected Done() = true after Cancel()")
	}

	c.Reset()
	if c.Done() {
		t.Error("expected Done() = false after Reset()")
	}
}

func TestContextCanceler_Context(t *testing.T) {
	parent := context.Background()
	c := cancel.NewContext(parent)

	ctx := c.Context()
	if ctx == nil {
		t.Error("expected non-nil context")
	}

	// Context should not be done yet
	select {
	case <-ctx.Done():
		t.Error("expected context to not be done")
	default:
		// OK
	}

	c.Cancel()

	// Context should be done now
	select {
	case <-ctx.Done():
		// OK
	default:
		t.Error("expected context to be done after Cancel()")
	}
}

// Test that both implementations satisfy the interface
func TestCancelerInterface(t *testing.T) {
	testCases := []struct {
		name string
		c    cancel.Canceler
	}{
		{"Context", cancel.NewContext(context.Background())},
		{"Atomic", cancel.NewAtomic()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.c.Done() {
				t.Error("expected Done() = false initially")
			}

			tc.c.Cancel()

			if !tc.c.Done() {
				t.Error("expected Done() = true after Cancel()")
			}
		})
	}
}
