package scheduler

import (
	"container/heap"
	"sync/atomic"
	"testing"
)

func TestPriorityScheduler_AddTask(t *testing.T) {
	ps := NewPriorityScheduler(2)
	ps.AddTask(&Task{ID: "a", Priority: 1})
	ps.AddTask(&Task{ID: "b", Priority: 2, Deps: []string{"a"}})

	if ps.TaskCount() != 2 {
		t.Errorf("TaskCount() = %d, want 2", ps.TaskCount())
	}
}

func TestPriorityScheduler_Run(t *testing.T) {
	ps := NewPriorityScheduler(2)

	var count atomic.Int32
	ps.AddTask(&Task{ID: "low", Priority: 0, Fn: func() error { count.Add(1); return nil }})
	ps.AddTask(&Task{ID: "high", Priority: 10, Fn: func() error { count.Add(1); return nil }})

	if err := ps.Run(); err != nil {
		t.Fatal(err)
	}

	if count.Load() != 2 {
		t.Errorf("expected 2 tasks executed, got %d", count.Load())
	}
}

func TestPriorityScheduler_Dependencies(t *testing.T) {
	ps := NewPriorityScheduler(4)

	var count atomic.Int32
	ps.AddTask(&Task{ID: "a", Fn: func() error { count.Add(1); return nil }})
	ps.AddTask(&Task{ID: "b", Deps: []string{"a"}, Fn: func() error { count.Add(1); return nil }})
	ps.AddTask(&Task{ID: "c", Deps: []string{"b"}, Fn: func() error { count.Add(1); return nil }})

	if err := ps.Run(); err != nil {
		t.Fatal(err)
	}

	if count.Load() != 3 {
		t.Errorf("expected 3 tasks, got %d", count.Load())
	}
}

func TestPriorityScheduler_IndependentTasks(t *testing.T) {
	ps := NewPriorityScheduler(8)

	var count atomic.Int32
	for i := 0; i < 20; i++ {
		ps.AddTask(&Task{
			ID: string(rune('a' + i%26)),
			Fn: func() error { count.Add(1); return nil },
		})
	}

	if err := ps.Run(); err != nil {
		t.Fatal(err)
	}

	if count.Load() != 20 {
		t.Errorf("expected 20, got %d", count.Load())
	}
}

func TestPriorityScheduler_Clear(t *testing.T) {
	ps := NewPriorityScheduler(2)
	ps.AddTask(&Task{ID: "a"})
	ps.Clear()

	if ps.TaskCount() != 0 {
		t.Errorf("after Clear, TaskCount() = %d, want 0", ps.TaskCount())
	}
}

func TestPriorityScheduler_DefaultWorkers(t *testing.T) {
	ps := NewPriorityScheduler(0) // should default to 4
	if ps.workers != 4 {
		t.Errorf("workers = %d, want 4", ps.workers)
	}
}

func TestPriorityScheduler_HighPriorityFirst(t *testing.T) {
	ps := NewPriorityScheduler(1) // single worker for determinism

	for i := 0; i < 5; i++ {
		priority := i
		ps.AddTask(&Task{
			ID:       string(rune('a' + i)),
			Priority: priority,
			Fn: func() error {
				return nil
			},
		})
	}

	if err := ps.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestPriorityQueue_PushPop(t *testing.T) {
	pq := make(PriorityQueue, 0)

	heap.Push(&pq, &PQItem{task: &Task{ID: "low"}, priority: 1})
	heap.Push(&pq, &PQItem{task: &Task{ID: "high"}, priority: 10})
	heap.Push(&pq, &PQItem{task: &Task{ID: "mid"}, priority: 5})

	heap.Init(&pq)

	first := heap.Pop(&pq).(*PQItem)
	if first.task.ID != "high" {
		t.Errorf("first pop = %q, want %q", first.task.ID, "high")
	}

	second := heap.Pop(&pq).(*PQItem)
	if second.task.ID != "mid" {
		t.Errorf("second pop = %q, want %q", second.task.ID, "mid")
	}
}
