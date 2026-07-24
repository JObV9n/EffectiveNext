package observability

import (
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
}

func TestRecord(t *testing.T) {
	c := NewCollector()
	c.Record("test.metric", 42.0, "count")

	metrics := c.Get("test.metric")
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Value != 42.0 {
		t.Errorf("expected value 42.0, got %f", metrics[0].Value)
	}
	if metrics[0].Unit != "count" {
		t.Errorf("expected unit count, got %s", metrics[0].Unit)
	}
}

func TestRecordWithTags(t *testing.T) {
	c := NewCollector()
	tags := map[string]string{"env": "test", "service": "effective-next"}
	c.RecordWithTags("test.metric", 100.0, "ms", tags)

	metrics := c.Get("test.metric")
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Tags["env"] != "test" {
		t.Errorf("expected env=test, got %s", metrics[0].Tags["env"])
	}
}

func TestGet_NonExistent(t *testing.T) {
	c := NewCollector()
	metrics := c.Get("nonexistent")
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestGetAll(t *testing.T) {
	c := NewCollector()
	c.Record("metric.a", 1.0, "count")
	c.Record("metric.b", 2.0, "count")
	c.Record("metric.a", 3.0, "count")

	all := c.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(all))
	}
}

func TestCount(t *testing.T) {
	c := NewCollector()
	if c.Count() != 0 {
		t.Errorf("expected 0, got %d", c.Count())
	}

	c.Record("test", 1.0, "count")
	if c.Count() != 1 {
		t.Errorf("expected 1, got %d", c.Count())
	}
}

func TestReset(t *testing.T) {
	c := NewCollector()
	c.Record("test", 1.0, "count")
	c.Reset()

	if c.Count() != 0 {
		t.Errorf("expected 0 after reset, got %d", c.Count())
	}
}

func TestTimer(t *testing.T) {
	c := NewCollector()
	timer := c.StartTimer("test.timer")

	time.Sleep(10 * time.Millisecond)
	timer.Stop()

	metrics := c.Get("test.timer")
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Value < 10 {
		t.Errorf("expected at least 10ms, got %f", metrics[0].Value)
	}
	if metrics[0].Unit != "ms" {
		t.Errorf("expected unit ms, got %s", metrics[0].Unit)
	}
}

func TestBuildTracker(t *testing.T) {
	bt := NewBuildTracker()
	if bt == nil {
		t.Fatal("expected non-nil tracker")
	}
}

func TestBuildTracker_RecordScan(t *testing.T) {
	bt := NewBuildTracker()
	bt.RecordScan(100, 500*time.Millisecond)

	summary := bt.Summary()
	if summary.TotalFiles != 100 {
		t.Errorf("expected 100 files, got %d", summary.TotalFiles)
	}
	if summary.ScanDurationMs != 500 {
		t.Errorf("expected 500ms, got %f", summary.ScanDurationMs)
	}
}

func TestBuildTracker_RecordBuild(t *testing.T) {
	bt := NewBuildTracker()
	bt.RecordBuild(10, 5, 2000*time.Millisecond)

	summary := bt.Summary()
	if summary.ChangedFiles != 10 {
		t.Errorf("expected 10 changed, got %d", summary.ChangedFiles)
	}
	if summary.RebuiltFiles != 5 {
		t.Errorf("expected 5 rebuilt, got %d", summary.RebuiltFiles)
	}
	if summary.BuildDurationMs != 2000 {
		t.Errorf("expected 2000ms, got %f", summary.BuildDurationMs)
	}
}

func TestBuildTracker_RecordCache(t *testing.T) {
	bt := NewBuildTracker()
	bt.RecordCache(80, 20)

	summary := bt.Summary()
	if summary.CacheHits != 80 {
		t.Errorf("expected 80 hits, got %d", summary.CacheHits)
	}
	if summary.CacheMisses != 20 {
		t.Errorf("expected 20 misses, got %d", summary.CacheMisses)
	}
	if summary.CacheHitRatio != 0.8 {
		t.Errorf("expected ratio 0.8, got %f", summary.CacheHitRatio)
	}
}

func TestBuildTracker_RecordCache_Zero(t *testing.T) {
	bt := NewBuildTracker()
	bt.RecordCache(0, 0)

	summary := bt.Summary()
	if summary.CacheHitRatio != 0 {
		t.Errorf("expected ratio 0, got %f", summary.CacheHitRatio)
	}
}

func TestBuildTracker_Summary(t *testing.T) {
	bt := NewBuildTracker()
	bt.RecordScan(100, 500*time.Millisecond)
	bt.RecordBuild(10, 5, 2000*time.Millisecond)

	summary := bt.Summary()
	if summary.TotalDurationMs != 2500 {
		t.Errorf("expected total 2500ms, got %f", summary.TotalDurationMs)
	}
}

func TestBuildTracker_Collector(t *testing.T) {
	bt := NewBuildTracker()
	c := bt.Collector()
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
}

func TestBuildSummary_String(t *testing.T) {
	summary := BuildSummary{
		TotalFiles:      100,
		ChangedFiles:    10,
		RebuiltFiles:    5,
		CacheHitRatio:   0.8,
		ScanDurationMs:  500,
		BuildDurationMs: 2000,
		TotalDurationMs: 2500,
		PeakMemoryMB:    128.5,
	}

	s := summary.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestMetric_Timestamp(t *testing.T) {
	c := NewCollector()
	before := time.Now()
	c.Record("test", 1.0, "count")
	after := time.Now()

	metrics := c.Get("test")
	if len(metrics) != 1 {
		t.Fatal("expected 1 metric")
	}

	if metrics[0].Timestamp.Before(before) || metrics[0].Timestamp.After(after) {
		t.Error("timestamp out of expected range")
	}
}

func TestConcurrentRecord(t *testing.T) {
	c := NewCollector()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				c.Record("test", float64(j), "count")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if c.Count() != 1000 {
		t.Errorf("expected 1000 metrics, got %d", c.Count())
	}
}

func TestMultipleTimers(t *testing.T) {
	c := NewCollector()

	t1 := c.StartTimer("timer1")
	t2 := c.StartTimer("timer2")

	time.Sleep(5 * time.Millisecond)
	t1.Stop()
	time.Sleep(5 * time.Millisecond)
	t2.Stop()

	m1 := c.Get("timer1")
	m2 := c.Get("timer2")

	if len(m1) != 1 || len(m2) != 1 {
		t.Fatal("expected 1 metric for each timer")
	}

	if m1[0].Value >= m2[0].Value {
		t.Error("expected timer1 to have shorter duration than timer2")
	}
}

func TestReset_ClearsAll(t *testing.T) {
	c := NewCollector()
	c.Record("a", 1.0, "count")
	c.Record("b", 2.0, "count")
	c.Reset()

	if len(c.GetAll()) != 0 {
		t.Error("expected no metrics after reset")
	}
}
