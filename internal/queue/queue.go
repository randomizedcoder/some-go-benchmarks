// Package queue provides SPSC queue implementations for benchmarking.
//
// This package offers two implementations of the Queue interface:
//   - ChannelQueue: Standard library approach using buffered channels
//   - RingBuffer: Optimized lock-free ring buffer
//
// # RingBuffer Safety (IMPORTANT)
//
// RingBuffer is a Single-Producer Single-Consumer (SPSC) queue.
// It is NOT safe for multiple goroutines to call Push() or Pop() concurrently.
//
// The implementation includes runtime guards that panic on misuse.
// This catches bugs early but adds ~1-2ns overhead per operation.
//
// Correct usage:
//   - Exactly ONE goroutine calls Push()
//   - Exactly ONE goroutine calls Pop()
//   - These may be the same goroutine or different goroutines
package queue

// Queue is a single-producer single-consumer queue.
//
// Implementations are non-blocking: Push returns false if full,
// Pop returns false if empty.
type Queue[T any] interface {
	// Push adds an item to the queue.
	// Returns false if the queue is full.
	Push(T) bool

	// Pop removes and returns an item from the queue.
	// Returns false if the queue is empty.
	Pop() (T, bool)
}
