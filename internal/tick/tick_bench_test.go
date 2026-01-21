package tick_test

import (
	"testing"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

// Long interval so Tick() returns false (we're measuring check overhead)
const benchInterval = time.Hour

// Sink variable to prevent compiler from eliminating benchmark loops
var sinkTick bool

// Direct type benchmarks (true performance floor)

func BenchmarkTick_Std_Direct(b *testing.B) {
	t := tick.NewTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

func BenchmarkTick_Batch_Direct(b *testing.B) {
	t := tick.NewBatch(benchInterval, 1000)
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

func BenchmarkTick_Atomic_Direct(b *testing.B) {
	t := tick.NewAtomicTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

// Interface benchmarks (with dynamic dispatch overhead)

func BenchmarkTick_Std_Interface(b *testing.B) {
	var t tick.Ticker = tick.NewTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

func BenchmarkTick_Atomic_Interface(b *testing.B) {
	var t tick.Ticker = tick.NewAtomicTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

// Reset benchmarks

func BenchmarkTick_Std_Reset(b *testing.B) {
	t := tick.NewTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Reset()
	}
}

func BenchmarkTick_Atomic_Reset(b *testing.B) {
	t := tick.NewAtomicTicker(benchInterval)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Reset()
	}
}

// Parallel benchmarks

func BenchmarkTick_Std_Parallel(b *testing.B) {
	t := tick.NewTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var result bool
		for pb.Next() {
			result = t.Tick()
		}
		sinkTick = result
	})
}

func BenchmarkTick_Atomic_Parallel(b *testing.B) {
	t := tick.NewAtomicTicker(benchInterval)
	defer t.Stop()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var result bool
		for pb.Next() {
			result = t.Tick()
		}
		sinkTick = result
	})
}
