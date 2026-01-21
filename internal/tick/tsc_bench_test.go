//go:build amd64

package tick_test

import (
	"testing"
	"time"

	"github.com/randomizedcoder/some-go-benchmarks/internal/tick"
)

func BenchmarkTick_TSC_Direct(b *testing.B) {
	t := tick.NewTSCCalibrated(time.Hour)
	b.ReportAllocs()
	b.ResetTimer()

	var result bool
	for i := 0; i < b.N; i++ {
		result = t.Tick()
	}
	sinkTick = result
}

func BenchmarkTick_TSC_Reset(b *testing.B) {
	t := tick.NewTSCCalibrated(time.Hour)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t.Reset()
	}
}

func BenchmarkCalibrateTSC(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	var result float64
	for i := 0; i < b.N; i++ {
		result = tick.CalibrateTSC()
	}
	_ = result
}
