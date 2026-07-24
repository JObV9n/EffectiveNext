package scheduler

import (
	"sync/atomic"
	"testing"
)

func TestScheduler_AddTask(t *testing.T) {
	s := New()
	s.AddTask(&Task{ID: "a", Name: "task-a"})
	s.AddTask(&Task{ID: "b", Name: "task-b", Deps: []string{"a"}})

	if s.TaskCount() != 2 {
		t.Errorf("TaskCount() = %d, want 2", s.TaskCount())
	}
}

func TestScheduler_Run(t *testing.T) {
	s := New()

	var count atomic.Int32
	s.AddTask(&Task{ID: "a", Fn: func() error { count.Add(1); return nil }})
	s.AddTask(&Task{ID: "b", Deps: []string{"a"}, Fn: func() error { count.Add(1); return nil }})
	s.AddTask(&Task{ID: "c", Deps: []string{"b"}, Fn: func() error { count.Add(1); return nil }})

	if err := s.Run(); err != nil {
		t.Fatal(err)
	}

	if count.Load() != 3 {
		t.Fatalf("expected 3 tasks executed, got %d", count.Load())
	}
}

func TestScheduler_IndependentTasks(t *testing.T) {
	s := New()

	var count atomic.Int32
	for i := 0; i < 10; i++ {
		s.AddTask(&Task{
			ID: string(rune('a' + i)),
			Fn: func() error { count.Add(1); return nil },
		})
	}

	if err := s.Run(); err != nil {
		t.Fatal(err)
	}

	if count.Load() != 10 {
		t.Errorf("expected 10 tasks, got %d", count.Load())
	}
}

func TestScheduler_Clear(t *testing.T) {
	s := New()
	s.AddTask(&Task{ID: "a"})
	s.Clear()

	if s.TaskCount() != 0 {
		t.Errorf("after Clear, TaskCount() = %d, want 0", s.TaskCount())
	}
}
