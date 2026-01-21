package queue

import (
	"sync/atomic"
)

// RingBuffer is a lock-free SPSC (Single-Producer Single-Consumer) queue.
//
// WARNING: This queue is NOT safe for multiple producers or multiple consumers.
// Using it incorrectly will cause data races and undefined behavior.
//
// The implementation includes runtime guards that panic if the SPSC contract
// is violated. This catches bugs early during development.
type RingBuffer[T any] struct {
	buf  []T
	mask uint64

	// Cache line padding to prevent false sharing
	_pad0 [56]byte //nolint:unused

	head atomic.Uint64 // Written by producer, read by consumer

	_pad1 [56]byte //nolint:unused

	tail atomic.Uint64 // Written by consumer, read by producer

	_pad2 [56]byte //nolint:unused

	// SPSC guards: detect concurrent misuse
	pushActive atomic.Uint32
	popActive  atomic.Uint32
}

// NewRingBuffer creates a RingBuffer with the specified size.
// Size will be rounded up to the next power of 2.
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	// Round up to power of 2
	n := uint64(1)
	for n < uint64(size) {
		n <<= 1
	}

	return &RingBuffer[T]{
		buf:  make([]T, n),
		mask: n - 1,
	}
}

// Push adds an item to the queue.
// Returns false if the queue is full.
//
// SPSC CONTRACT: Only ONE goroutine may call Push().
func (r *RingBuffer[T]) Push(v T) bool {
	// SPSC guard: panic if concurrent Push detected
	if !r.pushActive.CompareAndSwap(0, 1) {
		panic("queue: concurrent Push on SPSC RingBuffer - only one producer allowed")
	}
	defer r.pushActive.Store(0)

	head := r.head.Load()
	tail := r.tail.Load()

	// Check if full
	if head-tail >= uint64(len(r.buf)) {
		return false
	}

	// Write value
	r.buf[head&r.mask] = v

	// Publish (store-release semantics via atomic)
	r.head.Store(head + 1)

	return true
}

// Pop removes and returns an item from the queue.
// Returns false if the queue is empty.
//
// SPSC CONTRACT: Only ONE goroutine may call Pop().
func (r *RingBuffer[T]) Pop() (T, bool) {
	// SPSC guard: panic if concurrent Pop detected
	if !r.popActive.CompareAndSwap(0, 1) {
		panic("queue: concurrent Pop on SPSC RingBuffer - only one consumer allowed")
	}
	defer r.popActive.Store(0)

	tail := r.tail.Load()
	head := r.head.Load()

	// Check if empty
	if tail >= head {
		var zero T
		return zero, false
	}

	// Read value
	v := r.buf[tail&r.mask]

	// Consume (store-release semantics via atomic)
	r.tail.Store(tail + 1)

	return v, true
}

// Len returns the current number of items in the queue.
// This is an approximation and may be slightly stale.
func (r *RingBuffer[T]) Len() int {
	head := r.head.Load()
	tail := r.tail.Load()
	return int(head - tail)
}

// Cap returns the capacity of the queue.
func (r *RingBuffer[T]) Cap() int {
	return len(r.buf)
}
