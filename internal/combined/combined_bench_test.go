package combined_test

import (
	"context"
	"testing"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
	"github.com/randomizedcoder/some-go-benchmarks/internal/queue"
	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

// Sink variables
var sinkInt int
var sinkBool bool

const benchInterval = time.Hour

// ============================================================================
// Combined Cancel + Tick benchmarks
// ============================================================================

// BenchmarkCombined_CancelTick_Standard measures the combined overhead
// of checking context cancellation and ticker using standard library.
func BenchmarkCombined_CancelTick_Standard(b *testing.B) {
	ctx := cancel.NewContext(context.Background())
	ticker := tick.NewTicker(benchInterval)
	defer ticker.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	var cancelled, ticked bool
	for i := 0; i < b.N; i++ {
		cancelled = ctx.Done()
		ticked = ticker.Tick()
	}
	sinkBool = cancelled || ticked
}

// BenchmarkCombined_CancelTick_Optimized measures the same operations
// using atomic-based implementations.
func BenchmarkCombined_CancelTick_Optimized(b *testing.B) {
	ctx := cancel.NewAtomic()
	ticker := tick.NewAtomicTicker(benchInterval)
	b.ReportAllocs()
	b.ResetTimer()

	var cancelled, ticked bool
	for i := 0; i < b.N; i++ {
		cancelled = ctx.Done()
		ticked = ticker.Tick()
	}
	sinkBool = cancelled || ticked
}

// ============================================================================
// Full loop benchmarks (cancel + tick + queue)
// ============================================================================

// BenchmarkCombined_FullLoop_Standard simulates a realistic hot loop:
// check cancellation, check tick, process message from queue.
func BenchmarkCombined_FullLoop_Standard(b *testing.B) {
	ctx := cancel.NewContext(context.Background())
	ticker := tick.NewTicker(benchInterval)
	q := queue.NewChannel[int](1024)
	defer ticker.Stop()

	// Pre-fill queue
	for i := 0; i < 1024; i++ {
		q.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok, cancelled, ticked bool
	for i := 0; i < b.N; i++ {
		cancelled = ctx.Done()
		ticked = ticker.Tick()
		val, ok = q.Pop()
		q.Push(val) // Recycle
	}
	sinkInt = val
	sinkBool = ok || cancelled || ticked
}

// BenchmarkCombined_FullLoop_Optimized uses all optimized implementations.
func BenchmarkCombined_FullLoop_Optimized(b *testing.B) {
	ctx := cancel.NewAtomic()
	ticker := tick.NewAtomicTicker(benchInterval)
	q := queue.NewRingBuffer[int](1024)

	// Pre-fill queue
	for i := 0; i < 1024; i++ {
		q.Push(i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	var val int
	var ok, cancelled, ticked bool
	for i := 0; i < b.N; i++ {
		cancelled = ctx.Done()
		ticked = ticker.Tick()
		val, ok = q.Pop()
		q.Push(val) // Recycle
	}
	sinkInt = val
	sinkBool = ok || cancelled || ticked
}

// ============================================================================
// Pipeline benchmarks (producer/consumer)
// ============================================================================

// BenchmarkPipeline_Channel benchmarks a 2-goroutine SPSC pipeline
// using buffered channels.
func BenchmarkPipeline_Channel(b *testing.B) {
	q := queue.NewChannel[int](1024)
	done := make(chan struct{})

	// Consumer goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				q.Pop()
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for !q.Push(i) {
			// Spin until push succeeds
		}
	}

	b.StopTimer()
	close(done)
}

// BenchmarkPipeline_RingBuffer benchmarks a 2-goroutine SPSC pipeline
// using the lock-free ring buffer.
func BenchmarkPipeline_RingBuffer(b *testing.B) {
	q := queue.NewRingBuffer[int](1024)
	done := make(chan struct{})

	// Consumer goroutine (single consumer - SPSC contract)
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				q.Pop()
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	// Producer (single producer - SPSC contract)
	for i := 0; i < b.N; i++ {
		for !q.Push(i) {
			// Spin until push succeeds
		}
	}

	b.StopTimer()
	close(done)
}
