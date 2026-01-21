package combined_test

import (
	"sync/atomic"
	"testing"

	ring "github.com/randomizedcoder/go-lock-free-ring"
)

// ============================================================================
// Comparison Benchmarks: Channel vs Our SPSC vs go-lock-free-ring (MPSC)
// ============================================================================
//
// KEY DIFFERENCE:
// - Our RingBuffer: SPSC (Single-Producer, Single-Consumer)
// - go-lock-free-ring: MPSC (Multi-Producer, Single-Consumer) with sharding
//
// The sharded MPSC design is optimized for multiple producers, not single.

var sinkAny any
var sinkOkLfr bool

// ============================================================================
// SPSC: 1 Producer → 1 Consumer (comparing apples to apples)
// ============================================================================

// Our unguarded SPSC ring buffer (for fair comparison)
type spscRing struct {
	buf  []int
	mask uint64
	head atomic.Uint64
	tail atomic.Uint64
}

func newSPSCRing(size int) *spscRing {
	n := uint64(1)
	for n < uint64(size) {
		n <<= 1
	}
	return &spscRing{buf: make([]int, n), mask: n - 1}
}

func (r *spscRing) Push(v int) bool {
	head := r.head.Load()
	tail := r.tail.Load()
	if head-tail >= uint64(len(r.buf)) {
		return false
	}
	r.buf[head&r.mask] = v
	r.head.Store(head + 1)
	return true
}

func (r *spscRing) Pop() (int, bool) {
	tail := r.tail.Load()
	head := r.head.Load()
	if tail >= head {
		return 0, false
	}
	v := r.buf[tail&r.mask]
	r.tail.Store(tail + 1)
	return v, true
}

// BenchmarkLFR_SPSC_Channel - baseline channel
func BenchmarkLFR_SPSC_Channel(b *testing.B) {
	ch := make(chan int, 1024)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ch:
			default:
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for {
			select {
			case ch <- i:
				goto sent
			default:
			}
		}
	sent:
	}
	b.StopTimer()
	close(done)
}

// BenchmarkLFR_SPSC_OurRing - our unguarded SPSC
func BenchmarkLFR_SPSC_OurRing(b *testing.B) {
	q := newSPSCRing(1024)
	done := make(chan struct{})

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for !q.Push(i) {
		}
	}
	b.StopTimer()
	close(done)
}

// BenchmarkLFR_SPSC_ShardedRing1 - go-lock-free-ring with 1 shard (SPSC-like)
func BenchmarkLFR_SPSC_ShardedRing1(b *testing.B) {
	r, _ := ring.NewShardedRing(1024, 1)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				r.TryRead()
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for !r.Write(0, i) {
		}
	}
	b.StopTimer()
	close(done)
}

// ============================================================================
// MPSC: N Producers → 1 Consumer (where go-lock-free-ring shines)
// ============================================================================

// BenchmarkLFR_MPSC_Channel_4P - 4 producers using channel
func BenchmarkLFR_MPSC_Channel_4P(b *testing.B) {
	ch := make(chan int, 1024)
	done := make(chan struct{})
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for {
			select {
			case <-done:
				return
			case <-ch:
			default:
			}
		}
	}()

	b.SetParallelism(4)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for {
				select {
				case ch <- i:
					goto sent
				default:
				}
			}
		sent:
			i++
		}
	})

	b.StopTimer()
	close(done)
	<-consumerDone
}

// BenchmarkLFR_MPSC_ShardedRing_4P_4S - 4 producers, 4 shards
func BenchmarkLFR_MPSC_ShardedRing_4P_4S(b *testing.B) {
	r, _ := ring.NewShardedRing(1024, 4)
	done := make(chan struct{})
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for {
			select {
			case <-done:
				return
			default:
				r.TryRead()
			}
		}
	}()

	var producerID atomic.Uint64
	b.SetParallelism(4)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		pid := producerID.Add(1) - 1
		i := 0
		for pb.Next() {
			for !r.Write(pid, i) {
			}
			i++
		}
	})

	b.StopTimer()
	close(done)
	<-consumerDone
}

// BenchmarkLFR_MPSC_Channel_8P - 8 producers using channel
func BenchmarkLFR_MPSC_Channel_8P(b *testing.B) {
	ch := make(chan int, 1024)
	done := make(chan struct{})
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for {
			select {
			case <-done:
				return
			case <-ch:
			default:
			}
		}
	}()

	b.SetParallelism(8)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			for {
				select {
				case ch <- i:
					goto sent
				default:
				}
			}
		sent:
			i++
		}
	})

	b.StopTimer()
	close(done)
	<-consumerDone
}

// BenchmarkLFR_MPSC_ShardedRing_8P_8S - 8 producers, 8 shards
func BenchmarkLFR_MPSC_ShardedRing_8P_8S(b *testing.B) {
	r, _ := ring.NewShardedRing(2048, 8) // Larger capacity for 8 producers
	done := make(chan struct{})
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for {
			select {
			case <-done:
				return
			default:
				r.TryRead()
			}
		}
	}()

	var producerID atomic.Uint64
	b.SetParallelism(8)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		pid := producerID.Add(1) - 1
		i := 0
		for pb.Next() {
			for !r.Write(pid, i) {
			}
			i++
		}
	})

	b.StopTimer()
	close(done)
	<-consumerDone
}
