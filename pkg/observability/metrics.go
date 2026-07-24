package observability

import (
	"fmt"
	"sync"
	"time"
)

// Metric represents a single metric value.
type Metric struct {
	Name      string
	Value     float64
	Unit      string
	Timestamp time.Time
	Tags      map[string]string
}

// MetricsCollector collects and stores metrics.
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics []Metric
}

// NewCollector creates a new metrics collector.
func NewCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// Record records a metric.
func (mc *MetricsCollector) Record(name string, value float64, unit string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = append(mc.metrics, Metric{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
	})
}

// RecordWithTags records a metric with tags.
func (mc *MetricsCollector) RecordWithTags(name string, value float64, unit string, tags map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = append(mc.metrics, Metric{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Tags:      tags,
	})
}

// Get returns all metrics with the given name.
func (mc *MetricsCollector) Get(name string) []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	var result []Metric
	for _, m := range mc.metrics {
		if m.Name == name {
			result = append(result, m)
		}
	}
	return result
}

// GetAll returns all recorded metrics.
func (mc *MetricsCollector) GetAll() []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	result := make([]Metric, len(mc.metrics))
	copy(result, mc.metrics)
	return result
}

// Count returns the total number of recorded metrics.
func (mc *MetricsCollector) Count() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.metrics)
}

// Reset clears all recorded metrics.
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = nil
}

// Timer measures elapsed time.
type Timer struct {
	start time.Time
	name  string
	c     *MetricsCollector
}

// StartTimer begins a timer.
func (mc *MetricsCollector) StartTimer(name string) *Timer {
	return &Timer{
		start: time.Now(),
		name:  name,
		c:     mc,
	}
}

// Stop records the elapsed time.
func (t *Timer) Stop() {
	elapsed := time.Since(t.start)
	t.c.Record(t.name, float64(elapsed.Milliseconds()), "ms")
}

// BuildSummary holds a summary of build metrics.
type BuildSummary struct {
	TotalFiles     int     `json:"total_files"`
	ChangedFiles   int     `json:"changed_files"`
	RebuiltFiles   int     `json:"rebuilt_files"`
	CacheHits      int     `json:"cache_hits"`
	CacheMisses    int     `json:"cache_misses"`
	CacheHitRatio  float64 `json:"cache_hit_ratio"`
	ScanDurationMs float64 `json:"scan_duration_ms"`
	BuildDurationMs float64 `json:"build_duration_ms"`
	TotalDurationMs float64 `json:"total_duration_ms"`
	PeakMemoryMB   float64 `json:"peak_memory_mb"`
}

// String returns a human-readable build summary.
func (bs BuildSummary) String() string {
	return fmt.Sprintf(
		"files=%d changed=%d rebuilt=%d cache=%.1f%% scan=%sms build=%sms total=%sms mem=%.1fMB",
		bs.TotalFiles, bs.ChangedFiles, bs.RebuiltFiles,
		bs.CacheHitRatio*100,
		fmt.Sprintf("%.0f", bs.ScanDurationMs),
		fmt.Sprintf("%.0f", bs.BuildDurationMs),
		fmt.Sprintf("%.0f", bs.TotalDurationMs),
		bs.PeakMemoryMB,
	)
}

// BuildTracker tracks metrics for a single build.
type BuildTracker struct {
	collector *MetricsCollector
	summary   BuildSummary
}

// NewBuildTracker creates a new build tracker.
func NewBuildTracker() *BuildTracker {
	return &BuildTracker{
		collector: NewCollector(),
	}
}

// RecordScan records scan-related metrics.
func (bt *BuildTracker) RecordScan(fileCount int, duration time.Duration) {
	bt.summary.TotalFiles = fileCount
	bt.summary.ScanDurationMs = float64(duration.Milliseconds())
	bt.collector.Record("scan.files", float64(fileCount), "count")
	bt.collector.Record("scan.duration", float64(duration.Milliseconds()), "ms")
}

// RecordBuild records build-related metrics.
func (bt *BuildTracker) RecordBuild(changed, rebuilt int, duration time.Duration) {
	bt.summary.ChangedFiles = changed
	bt.summary.RebuiltFiles = rebuilt
	bt.summary.BuildDurationMs = float64(duration.Milliseconds())
	bt.collector.Record("build.changed", float64(changed), "count")
	bt.collector.Record("build.rebuilt", float64(rebuilt), "count")
	bt.collector.Record("build.duration", float64(duration.Milliseconds()), "ms")
}

// RecordCache records cache-related metrics.
func (bt *BuildTracker) RecordCache(hits, misses int) {
	bt.summary.CacheHits = hits
	bt.summary.CacheMisses = misses
	if hits+misses > 0 {
		bt.summary.CacheHitRatio = float64(hits) / float64(hits+misses)
	}
	bt.collector.Record("cache.hits", float64(hits), "count")
	bt.collector.Record("cache.misses", float64(misses), "count")
	bt.collector.Record("cache.ratio", bt.summary.CacheHitRatio, "ratio")
}

// Summary returns the build summary.
func (bt *BuildTracker) Summary() BuildSummary {
	bt.summary.TotalDurationMs = bt.summary.ScanDurationMs + bt.summary.BuildDurationMs
	return bt.summary
}

// Collector returns the underlying metrics collector.
func (bt *BuildTracker) Collector() *MetricsCollector {
	return bt.collector
}