package queue_test

import (
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

func testQueue[T comparable](t *testing.T, q queue.Queue[T], val T, name string) {
	t.Helper()

	// Empty queue returns false
	if _, ok := q.Pop(); ok {
		t.Errorf("%s: expected Pop() = false on empty queue", name)
	}

	// Push succeeds
	if !q.Push(val) {
		t.Errorf("%s: expected Push() = true", name)
	}

	// Pop returns pushed value
	got, ok := q.Pop()
	if !ok {
		t.Errorf("%s: expected Pop() = true after Push()", name)
	}
	if got != val {
		t.Errorf("%s: expected %v, got %v", name, val, got)
	}

	// Queue is empty again
	if _, ok := q.Pop(); ok {
		t.Errorf("%s: expected Pop() = false after draining", name)
	}
}

func TestChannelQueue(t *testing.T) {
	q := queue.NewChannel[int](8)
	testQueue(t, q, 42, "ChannelQueue")
}

func TestRingBuffer(t *testing.T) {
	q := queue.NewRingBuffer[int](8)
	testQueue(t, q, 42, "RingBuffer")
}

func TestChannelQueue_Full(t *testing.T) {
	q := queue.NewChannel[int](2)
	if !q.Push(1) {
		t.Error("expected Push(1) = true")
	}
	if !q.Push(2) {
		t.Error("expected Push(2) = true")
	}
	if q.Push(3) {
		t.Error("expected Push(3) = false on full queue")
	}
}

func TestRingBuffer_Full(t *testing.T) {
	q := queue.NewRingBuffer[int](2)
	if !q.Push(1) {
		t.Error("expected Push(1) = true")
	}
	if !q.Push(2) {
		t.Error("expected Push(2) = true")
	}
	if q.Push(3) {
		t.Error("expected Push(3) = false on full queue")
	}
}

func TestChannelQueue_FIFO(t *testing.T) {
	q := queue.NewChannel[int](8)

	for i := 0; i < 5; i++ {
		if !q.Push(i) {
			t.Fatalf("expected Push(%d) = true", i)
		}
	}

	for i := 0; i < 5; i++ {
		got, ok := q.Pop()
		if !ok {
			t.Fatalf("expected Pop() = true for item %d", i)
		}
		if got != i {
			t.Errorf("FIFO violation: expected %d, got %d", i, got)
		}
	}
}

func TestRingBuffer_FIFO(t *testing.T) {
	q := queue.NewRingBuffer[int](8)

	for i := 0; i < 5; i++ {
		if !q.Push(i) {
			t.Fatalf("expected Push(%d) = true", i)
		}
	}

	for i := 0; i < 5; i++ {
		got, ok := q.Pop()
		if !ok {
			t.Fatalf("expected Pop() = true for item %d", i)
		}
		if got != i {
			t.Errorf("FIFO violation: expected %d, got %d", i, got)
		}
	}
}

func TestChannelQueue_LenCap(t *testing.T) {
	q := queue.NewChannel[int](8)

	if q.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", q.Len())
	}
	if q.Cap() != 8 {
		t.Errorf("expected Cap() = 8, got %d", q.Cap())
	}

	q.Push(1)
	q.Push(2)

	if q.Len() != 2 {
		t.Errorf("expected Len() = 2, got %d", q.Len())
	}
}

func TestRingBuffer_LenCap(t *testing.T) {
	q := queue.NewRingBuffer[int](8)

	if q.Len() != 0 {
		t.Errorf("expected Len() = 0, got %d", q.Len())
	}
	if q.Cap() != 8 {
		t.Errorf("expected Cap() = 8, got %d", q.Cap())
	}

	q.Push(1)
	q.Push(2)

	if q.Len() != 2 {
		t.Errorf("expected Len() = 2, got %d", q.Len())
	}
}

func TestRingBuffer_PowerOfTwo(t *testing.T) {
	// Size 5 should round up to 8
	q := queue.NewRingBuffer[int](5)
	if q.Cap() != 8 {
		t.Errorf("expected Cap() = 8 (rounded up), got %d", q.Cap())
	}

	// Size 8 should stay 8
	q2 := queue.NewRingBuffer[int](8)
	if q2.Cap() != 8 {
		t.Errorf("expected Cap() = 8, got %d", q2.Cap())
	}
}

// Test that both implementations satisfy the interface
func TestQueueInterface(t *testing.T) {
	testCases := []struct {
		name string
		q    queue.Queue[int]
	}{
		{"Channel", queue.NewChannel[int](8)},
		{"RingBuffer", queue.NewRingBuffer[int](8)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testQueue(t, tc.q, 42, tc.name)
		})
	}
}
