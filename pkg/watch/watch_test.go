package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestNew(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 100)

	if w.WatchPath() != dir {
		t.Errorf("WatchPath() = %q, want %q", w.WatchPath(), dir)
	}
}

func TestNew_DefaultDebounce(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 0)

	if w.debounce != 100*time.Millisecond {
		t.Errorf("expected default debounce 100ms, got %v", w.debounce)
	}
}

func TestNew_CustomDebounce(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 250)

	if w.debounce != 250*time.Millisecond {
		t.Errorf("expected debounce 250ms, got %v", w.debounce)
	}
}

func TestEvents(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 100)

	ch := w.Events()
	if ch == nil {
		t.Error("Events() returned nil")
	}
}

func TestAddEvent(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 100)

	e := Event{
		Path: "/test.ts",
		Type: EventModify,
		Time: time.Now(),
	}

	w.AddEvent(e)

	select {
	case received := <-w.Events():
		if received.Path != e.Path {
			t.Errorf("expected path %q, got %q", e.Path, received.Path)
		}
		if received.Type != e.Type {
			t.Errorf("expected type %v, got %v", e.Type, received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestSetCallback(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 100)

	called := false
	w.SetCallback(func(e Event) {
		called = true
	})

	w.mu.Lock()
	w.callback_fn(Event{Path: "/test.ts", Type: EventModify})
	w.mu.Unlock()

	if !called {
		t.Error("callback was not called")
	}
}

func TestClassifyFSNotifyEvent(t *testing.T) {
	tests := []struct {
		name     string
		op       fsnotify.Op
		expected EventType
	}{
		{"create", fsnotify.Create, EventCreate},
		{"modify", fsnotify.Write, EventModify},
		{"delete", fsnotify.Remove, EventDelete},
		{"rename", fsnotify.Rename, EventRename},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := fsnotify.Event{Op: tt.op}
			result := classifyFSNotifyEvent(ev)
			if result != tt.expected {
				t.Errorf("classifyFSNotifyEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClassifyEvent(t *testing.T) {
	result := ClassifyEvent("/test.ts", "/")
	if result != EventModify {
		t.Errorf("ClassifyEvent() = %v, want %v", result, EventModify)
	}
}

func TestStartStop(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)

	w := New(dir, 100)
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	w.Stop()
}

func TestAddRecursive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub1"), 0o755)
	os.MkdirAll(filepath.Join(dir, "sub2"), 0o755)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)

	w := New(dir, 100)
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	watches := len(w.fsw.WatchList())
	if watches < 2 {
		t.Errorf("expected at least 2 watches, got %d", watches)
	}
}

func TestAddEvent_FullChannel(t *testing.T) {
	dir := t.TempDir()
	w := New(dir, 100)

	for i := 0; i < 100; i++ {
		w.AddEvent(Event{Path: "/test.ts", Type: EventModify})
	}

	w.AddEvent(Event{Path: "/overflow.ts", Type: EventModify})
}

func TestWatcherEventType(t *testing.T) {
	events := []EventType{EventCreate, EventModify, EventDelete, EventRename}
	for _, e := range events {
		if e < 0 {
			t.Errorf("unexpected negative event type: %v", e)
		}
	}
}
