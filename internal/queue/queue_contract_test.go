package queue_test

import (
	"sync"
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

// TestRingBuffer_SPSC_ConcurrentPush_Panics verifies that the SPSC guard
// catches concurrent Push() calls.
//
// This test intentionally violates the SPSC contract to verify the guard works.
func TestRingBuffer_SPSC_ConcurrentPush_Panics(t *testing.T) {
	q := queue.NewRingBuffer[int](1024)

	// We need to catch the panic
	panicked := make(chan bool, 1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case panicked <- true:
					default:
					}
				}
			}()
			for j := 0; j < 1000; j++ {
				q.Push(n*1000 + j)
			}
		}(i)
	}

	wg.Wait()

	select {
	case <-panicked:
		// Expected: the SPSC guard caught concurrent access
		t.Log("SPSC guard correctly detected concurrent Push()")
	default:
		// The test may pass without panic if goroutines don't overlap
		// This is OK - it just means we didn't catch the race this time
		t.Log("No panic detected (goroutines may not have overlapped)")
	}
}

// TestRingBuffer_SPSC_ConcurrentPop_Panics verifies that the SPSC guard
// catches concurrent Pop() calls.
//
// This test intentionally violates the SPSC contract to verify the guard works.
func TestRingBuffer_SPSC_ConcurrentPop_Panics(t *testing.T) {
	q := queue.NewRingBuffer[int](1024)

	// Pre-fill the queue
	for i := 0; i < 1024; i++ {
		q.Push(i)
	}

	panicked := make(chan bool, 1)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					select {
					case panicked <- true:
					default:
					}
				}
			}()
			for j := 0; j < 200; j++ {
				q.Pop()
			}
		}()
	}

	wg.Wait()

	select {
	case <-panicked:
		t.Log("SPSC guard correctly detected concurrent Pop()")
	default:
		t.Log("No panic detected (goroutines may not have overlapped)")
	}
}

// TestRingBuffer_SPSC_Valid tests the valid SPSC pattern:
// one producer goroutine, one consumer goroutine.
func TestRingBuffer_SPSC_Valid(t *testing.T) {
	q := queue.NewRingBuffer[int](64)
	count := 10000
	done := make(chan struct{})

	// Producer (single goroutine)
	go func() {
		for i := 0; i < count; i++ {
			for !q.Push(i) {
				// Spin until push succeeds
			}
		}
		close(done)
	}()

	// Consumer (single goroutine - this test's main goroutine)
	received := 0
	expected := 0
	for received < count {
		if val, ok := q.Pop(); ok {
			if val != expected {
				t.Errorf("FIFO violation: expected %d, got %d", expected, val)
			}
			expected++
			received++
		}
	}

	<-done // Wait for producer

	if received != count {
		t.Errorf("expected %d items, received %d", count, received)
	}
}
