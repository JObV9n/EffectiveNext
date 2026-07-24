package observability

import (
	"testing"
	"time"
)

func TestTracer_StartSpan(t *testing.T) {
	tracer := NewTracer()
	span := tracer.StartSpan("scan", nil)
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if span.Name != "scan" {
		t.Errorf("Name = %q, want %q", span.Name, "scan")
	}
}

func TestTracer_FinishSpan(t *testing.T) {
	tracer := NewTracer()
	span := tracer.StartSpan("build", nil)
	time.Sleep(5 * time.Millisecond)
	tracer.FinishSpan(span)

	if span.End.IsZero() {
		t.Error("expected End to be set")
	}
	if span.Duration() < 5*time.Millisecond {
		t.Errorf("Duration = %v, want >= 5ms", span.Duration())
	}
}

func TestTracer_Spans(t *testing.T) {
	tracer := NewTracer()
	tracer.StartSpan("a", nil)
	tracer.StartSpan("b", nil)

	spans := tracer.Spans()
	if len(spans) != 2 {
		t.Errorf("Spans() = %d, want 2", len(spans))
	}
}

func TestTracer_Clear(t *testing.T) {
	tracer := NewTracer()
	tracer.StartSpan("a", nil)
	tracer.Clear()

	if len(tracer.Spans()) != 0 {
		t.Error("expected 0 spans after clear")
	}
}

func TestTracer_WithTags(t *testing.T) {
	tracer := NewTracer()
	tags := map[string]string{"worker": "1", "phase": "parse"}
	span := tracer.StartSpan("parse", tags)

	if span.Tags["worker"] != "1" {
		t.Errorf("Tags[worker] = %q, want %q", span.Tags["worker"], "1")
	}
}

func TestSpan_Duration_Running(t *testing.T) {
	span := &Span{Start: time.Now()}
	d := span.Duration()
	if d <= 0 {
		t.Errorf("Duration for running span = %v, want > 0", d)
	}
}

func TestTimeline_BeginEnd(t *testing.T) {
	tl := NewTimeline()
	tl.Begin("scan", "scan", 0)
	time.Sleep(5 * time.Millisecond)
	tl.End("scan")

	events := tl.Events()
	if len(events) != 1 {
		t.Fatalf("Events() = %d, want 1", len(events))
	}
	if events[0].Duration < 5*time.Millisecond {
		t.Errorf("Duration = %v, want >= 5ms", events[0].Duration)
	}
}

func TestTimeline_Events(t *testing.T) {
	tl := NewTimeline()
	tl.Begin("a", "phase-a", 0)
	tl.Begin("b", "phase-b", 1)
	tl.End("a")

	events := tl.Events()
	if len(events) != 2 {
		t.Errorf("Events() = %d, want 2", len(events))
	}
}

func TestTimeline_TotalDuration(t *testing.T) {
	tl := NewTimeline()
	tl.Begin("a", "phase", 0)
	tl.Begin("b", "phase", 0)
	time.Sleep(5 * time.Millisecond)
	tl.End("a")
	tl.End("b")

	total := tl.TotalDuration()
	if total < 5*time.Millisecond {
		t.Errorf("TotalDuration = %v, want >= 5ms", total)
	}
}

func TestWorkerUtilization_RecordSample(t *testing.T) {
	wu := NewWorkerUtilization(4)
	wu.RecordSample(2)

	samples := wu.Samples()
	if len(samples) != 1 {
		t.Fatalf("Samples() = %d, want 1", len(samples))
	}
	if samples[0].BusyWorkers != 2 {
		t.Errorf("BusyWorkers = %d, want 2", samples[0].BusyWorkers)
	}
	if samples[0].IdleWorkers != 2 {
		t.Errorf("IdleWorkers = %d, want 2", samples[0].IdleWorkers)
	}
}

func TestWorkerUtilization_Average(t *testing.T) {
	wu := NewWorkerUtilization(4)
	wu.RecordSample(4) // 100%
	wu.RecordSample(0) // 0%

	avg := wu.AverageUtilization()
	if avg != 0.5 {
		t.Errorf("AverageUtilization = %f, want 0.5", avg)
	}
}

func TestWorkerUtilization_AverageEmpty(t *testing.T) {
	wu := NewWorkerUtilization(4)
	avg := wu.AverageUtilization()
	if avg != 0 {
		t.Errorf("AverageUtilization = %f, want 0", avg)
	}
}

func TestWorkerUtilization_Overcount(t *testing.T) {
	wu := NewWorkerUtilization(2)
	wu.RecordSample(10) // more than total

	samples := wu.Samples()
	if samples[0].BusyWorkers != 2 {
		t.Errorf("BusyWorkers = %d, want capped at 2", samples[0].BusyWorkers)
	}
}

func TestCaptureProfile(t *testing.T) {
	p := CaptureProfile()
	if p.Goroutines <= 0 {
		t.Error("expected Goroutines > 0")
	}
	if p.MemoryMB <= 0 {
		t.Error("expected MemoryMB > 0")
	}
}

func TestMemoryTracker_Snapshot(t *testing.T) {
	mt := NewMemoryTracker()
	p := mt.Snapshot()
	if p.Goroutines <= 0 {
		t.Error("expected non-zero goroutines")
	}

	snapshots := mt.Snapshots()
	if len(snapshots) != 1 {
		t.Errorf("Snapshots() = %d, want 1", len(snapshots))
	}
}

func TestMemoryTracker_Peak(t *testing.T) {
	mt := NewMemoryTracker()
	mt.Snapshot()
	p := mt.Peak()

	if p.HeapAllocMB < 0 {
		t.Error("expected non-negative peak")
	}
}

func TestTracer_Concurrent(t *testing.T) {
	tracer := NewTracer()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			s := tracer.StartSpan("concurrent", nil)
			tracer.FinishSpan(s)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	spans := tracer.Spans()
	if len(spans) != 10 {
		t.Errorf("Spans() = %d, want 10", len(spans))
	}
}
