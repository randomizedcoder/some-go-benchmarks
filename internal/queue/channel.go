package queue

// ChannelQueue wraps a buffered channel as a Queue.
//
// This is the standard library approach. Each Push/Pop performs
// a non-blocking channel operation via select with default.
type ChannelQueue[T any] struct {
	ch chan T
}

// NewChannel creates a ChannelQueue with the specified buffer size.
func NewChannel[T any](size int) *ChannelQueue[T] {
	return &ChannelQueue[T]{
		ch: make(chan T, size),
	}
}

// Push adds an item to the queue.
// Returns false if the queue is full (non-blocking).
func (q *ChannelQueue[T]) Push(v T) bool {
	select {
	case q.ch <- v:
		return true
	default:
		return false
	}
}

// Pop removes and returns an item from the queue.
// Returns false if the queue is empty (non-blocking).
func (q *ChannelQueue[T]) Pop() (T, bool) {
	select {
	case v := <-q.ch:
		return v, true
	default:
		var zero T
		return zero, false
	}
}

// Len returns the current number of items in the queue.
func (q *ChannelQueue[T]) Len() int {
	return len(q.ch)
}

// Cap returns the capacity of the queue.
func (q *ChannelQueue[T]) Cap() int {
	return cap(q.ch)
}
