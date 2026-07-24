package observability

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Span represents a traced unit of work.
type Span struct {
	Name     string
	Start    time.Time
	End      time.Time
	Tags     map[string]string
	ParentID string
	ID       string
}

// Duration returns the span's elapsed time.
func (s *Span) Duration() time.Duration {
	if s.End.IsZero() {
		return time.Since(s.Start)
	}
	return s.End.Sub(s.Start)
}

// Tracer records execution traces for build operations.
type Tracer struct {
	mu    sync.RWMutex
	spans []*Span
}

// NewTracer creates a new trace collector.
func NewTracer() *Tracer {
	return &Tracer{}
}

// StartSpan begins a new span and returns it.
func (t *Tracer) StartSpan(name string, tags map[string]string) *Span {
	span := &Span{
		Name:  name,
		Start: time.Now(),
		Tags:  tags,
		ID:    fmt.Sprintf("span-%d", time.Now().UnixNano()),
	}
	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()
	return span
}

// FinishSpan marks a span as complete.
func (t *Tracer) FinishSpan(span *Span) {
	span.End = time.Now()
}

// Spans returns all recorded spans.
func (t *Tracer) Spans() []*Span {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*Span, len(t.spans))
	copy(result, t.spans)
	return result
}

// Clear removes all recorded spans.
func (t *Tracer) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = nil
}

// Timeline represents a chronological timeline of build phases.
type Timeline struct {
	mu     sync.Mutex
	events []TimelineEvent
}

// TimelineEvent represents a single event in the build timeline.
type TimelineEvent struct {
	Name      string        `json:"name"`
	Phase     string        `json:"phase"`
	Start     time.Time     `json:"start"`
	Duration  time.Duration `json:"duration"`
	WorkerID  int           `json:"worker_id"`
}

// NewTimeline creates a new build timeline.
func NewTimeline() *Timeline {
	return &Timeline{}
}

// Begin starts tracking a phase.
func (tl *Timeline) Begin(name, phase string, workerID int) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	tl.events = append(tl.events, TimelineEvent{
		Name:     name,
		Phase:    phase,
		Start:    time.Now(),
		WorkerID: workerID,
	})
}

// End closes the most recent event with the given name.
func (tl *Timeline) End(name string) {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	for i := len(tl.events) - 1; i >= 0; i-- {
		if tl.events[i].Name == name && tl.events[i].Duration == 0 {
			tl.events[i].Duration = time.Since(tl.events[i].Start)
			return
		}
	}
}

// Events returns all timeline events.
func (tl *Timeline) Events() []TimelineEvent {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	result := make([]TimelineEvent, len(tl.events))
	copy(result, tl.events)
	return result
}

// TotalDuration returns the sum of all event durations.
func (tl *Timeline) TotalDuration() time.Duration {
	tl.mu.Lock()
	defer tl.mu.Unlock()
	var total time.Duration
	for _, e := range tl.events {
		total += e.Duration
	}
	return total
}

// WorkerUtilization tracks worker thread utilization over time.
type WorkerUtilization struct {
	mu          sync.Mutex
	samples     []WorkerSample
	totalWorkers int
}

// WorkerSample represents a utilization snapshot.
type WorkerSample struct {
	Timestamp   time.Time `json:"timestamp"`
	BusyWorkers int       `json:"busy_workers"`
	IdleWorkers int       `json:"idle_workers"`
	Utilization float64   `json:"utilization"`
}

// NewWorkerUtilization creates a new utilization tracker.
func NewWorkerUtilization(totalWorkers int) *WorkerUtilization {
	return &WorkerUtilization{
		totalWorkers: totalWorkers,
	}
}

// RecordSample records a point-in-time utilization sample.
func (wu *WorkerUtilization) RecordSample(busyWorkers int) {
	wu.mu.Lock()
	defer wu.mu.Unlock()

	idle := wu.totalWorkers - busyWorkers
	if idle < 0 {
		idle = 0
		busyWorkers = wu.totalWorkers
	}

	var util float64
	if wu.totalWorkers > 0 {
		util = float64(busyWorkers) / float64(wu.totalWorkers)
	}

	wu.samples = append(wu.samples, WorkerSample{
		Timestamp:   time.Now(),
		BusyWorkers: busyWorkers,
		IdleWorkers: idle,
		Utilization: util,
	})
}

// AverageUtilization returns the mean utilization across all samples.
func (wu *WorkerUtilization) AverageUtilization() float64 {
	wu.mu.Lock()
	defer wu.mu.Unlock()

	if len(wu.samples) == 0 {
		return 0
	}

	var total float64
	for _, s := range wu.samples {
		total += s.Utilization
	}
	return total / float64(len(wu.samples))
}

// Samples returns all recorded samples.
func (wu *WorkerUtilization) Samples() []WorkerSample {
	wu.mu.Lock()
	defer wu.mu.Unlock()
	result := make([]WorkerSample, len(wu.samples))
	copy(result, wu.samples)
	return result
}

// Profile captures memory and CPU statistics.
type Profile struct {
	MemoryMB    float64 `json:"memory_mb"`
	HeapAllocMB float64 `json:"heap_alloc_mb"`
	HeapObjects int     `json:"heap_objects"`
	Goroutines  int     `json:"goroutines"`
	GCCycles    uint32  `json:"gc_cycles"`
}

// CaptureProfile takes a snapshot of current runtime statistics.
func CaptureProfile() Profile {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return Profile{
		MemoryMB:    float64(memStats.Sys) / 1024 / 1024,
		HeapAllocMB: float64(memStats.HeapAlloc) / 1024 / 1024,
		HeapObjects: int(memStats.HeapObjects),
		Goroutines:  runtime.NumGoroutine(),
		GCCycles:    memStats.NumGC,
	}
}

// MemoryTracker records periodic memory snapshots to find peak usage.
type MemoryTracker struct {
	mu       sync.Mutex
	snapshots []Profile
	peak     Profile
}

// NewMemoryTracker creates a new memory tracker.
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{}
}

// Snapshot captures a memory profile and tracks peak usage.
func (mt *MemoryTracker) Snapshot() Profile {
	p := CaptureProfile()
	mt.mu.Lock()
	mt.snapshots = append(mt.snapshots, p)
	if p.HeapAllocMB > mt.peak.HeapAllocMB {
		mt.peak = p
	}
	mt.mu.Unlock()
	return p
}

// Peak returns the snapshot with highest heap usage.
func (mt *MemoryTracker) Peak() Profile {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return mt.peak
}

// Snapshots returns all recorded profiles.
func (mt *MemoryTracker) Snapshots() []Profile {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	result := make([]Profile, len(mt.snapshots))
	copy(result, mt.snapshots)
	return result
}
