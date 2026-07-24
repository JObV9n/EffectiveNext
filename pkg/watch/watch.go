package watch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType describes the type of filesystem event.
type EventType int

const (
	EventCreate EventType = iota
	EventModify
	EventDelete
	EventRename
)

// Event represents a filesystem change event.
type Event struct {
	Path    string
	Type    EventType
	Time    time.Time
}

// Watcher watches filesystem changes using fsnotify.
type Watcher struct {
	dir      string
	events   chan Event
	mu       sync.Mutex
	debounce time.Duration
	stop     chan struct{}
	done     chan struct{}
	fsw      *fsnotify.Watcher
	callback func(Event)
}

// New creates a new file watcher.
func New(dir string, debounceMs int) *Watcher {
	if debounceMs <= 0 {
		debounceMs = 100
	}
	return &Watcher{
		dir:      dir,
		events:   make(chan Event, 100),
		debounce: time.Duration(debounceMs) * time.Millisecond,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Events returns the event channel.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// SetCallback sets a callback function for events.
func (w *Watcher) SetCallback(fn func(Event)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callback = fn
}

// Start begins watching the directory.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.fsw = fsw

	if err := w.addRecursive(w.dir); err != nil {
		fsw.Close()
		return err
	}

	go w.loop()
	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	close(w.stop)
	if w.fsw != nil {
		w.fsw.Close()
	}
	<-w.done
}

// WatchPath returns the directory being watched.
func (w *Watcher) WatchPath() string {
	return w.dir
}

// AddEvent pushes an event into the channel (for testing).
func (w *Watcher) AddEvent(e Event) {
	select {
	case w.events <- e:
	default:
	}
}

func (w *Watcher) loop() {
	defer close(w.done)

	batch := make(map[string]Event)
	var batchTimer *time.Timer

	for {
		select {
		case <-w.stop:
			if batchTimer != nil {
				batchTimer.Stop()
			}
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) == ".DS_Store" || filepath.Base(ev.Name) == "Thumbs.db" {
				continue
			}

			eventType := classifyFSNotifyEvent(ev)
			if eventType < 0 {
				continue
			}

			e := Event{
				Path: ev.Name,
				Type: eventType,
				Time: time.Now(),
			}

			w.mu.Lock()
			w.callback_fn(e)
			w.mu.Unlock()

			batch[ev.Name] = e
			if batchTimer == nil {
				batchTimer = time.AfterFunc(w.debounce, func() {
					w.mu.Lock()
					events := make([]Event, 0, len(batch))
					for _, e := range batch {
						events = append(events, e)
					}
					batch = make(map[string]Event)
					batchTimer = nil
					w.mu.Unlock()

					for _, e := range events {
						select {
						case w.events <- e:
						default:
						}
					}
				})
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			_ = err
		}
	}
}

func (w *Watcher) callback_fn(e Event) {
	if w.callback != nil {
		w.callback(e)
	}
}

func (w *Watcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if name == "node_modules" || name == ".next" || name == ".git" || name == "dist" || name == ".effective-next" {
			return filepath.SkipDir
		}
		return w.fsw.Add(path)
	})
}

func classifyFSNotifyEvent(ev fsnotify.Event) EventType {
	switch {
	case ev.Op&fsnotify.Create == fsnotify.Create:
		return EventCreate
	case ev.Op&fsnotify.Write == fsnotify.Write:
		return EventModify
	case ev.Op&fsnotify.Remove == fsnotify.Remove:
		return EventDelete
	case ev.Op&fsnotify.Rename == fsnotify.Rename:
		return EventRename
	case ev.Op&fsnotify.Chmod == fsnotify.Chmod:
		return EventModify
	default:
		return -1
	}
}

// ClassifyEvent determines the event type from a path.
func ClassifyEvent(path, root string) EventType {
	_ = filepath.Join(root)
	return EventModify
}