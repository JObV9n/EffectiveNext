package scheduler

import (
	"container/heap"
	"fmt"
	"testing"
)

func BenchmarkPriorityScheduler_Run_10(b *testing.B) {
	benchScheduler(b, 10, 4)
}

func BenchmarkPriorityScheduler_Run_100(b *testing.B) {
	benchScheduler(b, 100, 4)
}

func BenchmarkPriorityScheduler_Run_1000(b *testing.B) {
	benchScheduler(b, 1000, 8)
}

func BenchmarkPriorityScheduler_Run_WithDeps(b *testing.B) {
	for _, workers := range []int{1, 2, 4, 8} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ps := NewPriorityScheduler(workers)
				for j := 0; j < 100; j++ {
					deps := []string{}
					if j > 0 {
						deps = []string{fmt.Sprintf("task-%d", j-1)}
					}
					ps.AddTask(&Task{
						ID:       fmt.Sprintf("task-%d", j),
						Priority: j,
						Deps:     deps,
						Fn:       func() error { return nil },
					})
				}
				ps.Run()
			}
		})
	}
}

func BenchmarkDAGScheduler_Run_100(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := New()
		for j := 0; j < 100; j++ {
			s.AddTask(&Task{
				ID: fmt.Sprintf("task-%d", j),
				Fn: func() error { return nil },
			})
		}
		s.Run()
	}
}

func benchScheduler(b *testing.B, taskCount, workers int) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ps := NewPriorityScheduler(workers)
		for j := 0; j < taskCount; j++ {
			ps.AddTask(&Task{
				ID:       fmt.Sprintf("task-%d", j),
				Priority: j,
				Fn:       func() error { return nil },
			})
		}
		ps.Run()
	}
}

func BenchmarkPriorityQueue_PushPop(b *testing.B) {
	pq := make(PriorityQueue, 0)
	tasks := make([]*PQItem, 1000)
	for i := 0; i < 1000; i++ {
		tasks[i] = &PQItem{task: &Task{ID: fmt.Sprintf("task-%d", i)}, priority: i}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq = pq[:0]
		for _, t := range tasks {
			heap.Push(&pq, t)
		}
		for pq.Len() > 0 {
			heap.Pop(&pq)
		}
	}
}
