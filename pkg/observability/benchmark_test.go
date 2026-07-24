package observability

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkTracer_StartFinish(b *testing.B) {
	tracer := NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span := tracer.StartSpan("op", nil)
		tracer.FinishSpan(span)
	}
}

func BenchmarkTracer_ConcurrentSpans(b *testing.B) {
	tracer := NewTracer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		done := make(chan bool, 10)
		for j := 0; j < 10; j++ {
			go func() {
				s := tracer.StartSpan("concurrent", nil)
				tracer.FinishSpan(s)
				done <- true
			}()
		}
		for j := 0; j < 10; j++ {
			<-done
		}
	}
}

func BenchmarkTimeline_BeginEnd(b *testing.B) {
	tl := NewTimeline()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tl.Begin(fmt.Sprintf("phase-%d", i%5), "build", i%4)
		tl.End(fmt.Sprintf("phase-%d", i%5))
	}
}

func BenchmarkWorkerUtilization_Record(b *testing.B) {
	wu := NewWorkerUtilization(8)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wu.RecordSample(i % 9)
	}
}

func BenchmarkCaptureProfile(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CaptureProfile()
	}
}

func BenchmarkMemoryTracker_Snapshot(b *testing.B) {
	mt := NewMemoryTracker()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mt.Snapshot()
	}
}

func BenchmarkBuildTracker_FullCycle(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bt := NewBuildTracker()
		bt.RecordScan(100, 50*time.Millisecond)
		bt.RecordBuild(10, 5, 200*time.Millisecond)
		bt.RecordCache(80, 20)
		bt.Summary()
	}
}

func BenchmarkMetricsCollector_RecordParallel(b *testing.B) {
	c := NewCollector()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Record("bench.metric", 1.0, "count")
		}
	})
}
