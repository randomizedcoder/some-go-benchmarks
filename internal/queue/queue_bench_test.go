package queue_test

import (
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/queue"
)

// Sink variables to prevent compiler from eliminating benchmark loops
var sinkInt int
var sinkBool bool

// Direct type benchmarks (true performance floor)

func BenchmarkQueue_Channel_PushPop_Direct(b *testing.B) {
	q := queue.NewChannel[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok bool
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, ok = q.Pop()
	}
	sinkInt = val
	sinkBool = ok
}

func BenchmarkQueue_RingBuffer_PushPop_Direct(b *testing.B) {
	q := queue.NewRingBuffer[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok bool
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, ok = q.Pop()
	}
	sinkInt = val
	sinkBool = ok
}

// Interface benchmarks (with dynamic dispatch overhead)

func BenchmarkQueue_Channel_PushPop_Interface(b *testing.B) {
	var q queue.Queue[int] = queue.NewChannel[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok bool
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, ok = q.Pop()
	}
	sinkInt = val
	sinkBool = ok
}

func BenchmarkQueue_RingBuffer_PushPop_Interface(b *testing.B) {
	var q queue.Queue[int] = queue.NewRingBuffer[int](1024)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok bool
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, ok = q.Pop()
	}
	sinkInt = val
	sinkBool = ok
}

// Push-only benchmarks

func BenchmarkQueue_Channel_Push(b *testing.B) {
	q := queue.NewChannel[int](b.N + 1)
	b.ReportAllocs()
	b.ResetTimer()

	var ok bool
	for i := 0; i < b.N; i++ {
		ok = q.Push(i)
	}
	sinkBool = ok
}

func BenchmarkQueue_RingBuffer_Push(b *testing.B) {
	// Ensure buffer is large enough
	size := b.N
	if size < 1024 {
		size = 1024
	}
	q := queue.NewRingBuffer[int](size)
	b.ReportAllocs()
	b.ResetTimer()

	var ok bool
	for i := 0; i < b.N; i++ {
		ok = q.Push(i)
	}
	sinkBool = ok
}

// Different queue sizes

func BenchmarkQueue_Channel_PushPop_Size64(b *testing.B) {
	q := queue.NewChannel[int](64)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, _ = q.Pop()
	}
	sinkInt = val
}

func BenchmarkQueue_RingBuffer_PushPop_Size64(b *testing.B) {
	q := queue.NewRingBuffer[int](64)
	b.ReportAllocs()
	b.ResetTimer()

	var val int
	for i := 0; i < b.N; i++ {
		q.Push(i)
		val, _ = q.Pop()
	}
	sinkInt = val
}
