package cancel_test

import (
	"context"
	"testing"

	"github.com/randomizedcoder/some-go-benchmarks/internal/cancel"
)

// Sink variables to prevent compiler from eliminating benchmark loops
var sinkBool bool

// Direct type benchmarks (true performance floor)

func BenchmarkCancel_Context_Done_Direct(b *testing.B) {
	c := cancel.NewContext(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = c.Done()
	}
	sinkBool = result
}

func BenchmarkCancel_Atomic_Done_Direct(b *testing.B) {
	c := cancel.NewAtomic()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = c.Done()
	}
	sinkBool = result
}

// Interface benchmarks (realistic usage with dynamic dispatch)

func BenchmarkCancel_Context_Done_Interface(b *testing.B) {
	var c cancel.Canceler = cancel.NewContext(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = c.Done()
	}
	sinkBool = result
}

func BenchmarkCancel_Atomic_Done_Interface(b *testing.B) {
	var c cancel.Canceler = cancel.NewAtomic()
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = c.Done()
	}
	sinkBool = result
}

// Parallel benchmarks (multiple goroutines checking)

func BenchmarkCancel_Context_Done_Parallel(b *testing.B) {
	c := cancel.NewContext(context.Background())
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var result bool
		for pb.Next() {
			result = c.Done()
		}
		sinkBool = result
	})
}

func BenchmarkCancel_Atomic_Done_Parallel(b *testing.B) {
	c := cancel.NewAtomic()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		var result bool
		for pb.Next() {
			result = c.Done()
		}
		sinkBool = result
	})
}

// Reset benchmark
func BenchmarkCancel_Atomic_Reset(b *testing.B) {
	c := cancel.NewAtomic()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Reset()
	}
}
