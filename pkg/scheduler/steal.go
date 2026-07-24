package scheduler

import (
	"container/heap"
	"sync"
	"sync/atomic"
)

// PriorityScheduler is a work-stealing task scheduler with priority queuing.
type PriorityScheduler struct {
	tasks    map[string]*Task
	inDegree map[string]int
	mu       sync.Mutex
	workers  int
}

// NewPriorityScheduler creates a new priority-aware scheduler.
func NewPriorityScheduler(workers int) *PriorityScheduler {
	if workers <= 0 {
		workers = 4
	}
	return &PriorityScheduler{
		tasks:    make(map[string]*Task),
		inDegree: make(map[string]int),
		workers:  workers,
	}
}

// AddTask adds a task to the scheduler.
func (ps *PriorityScheduler) AddTask(task *Task) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.tasks[task.ID] = task
	ps.inDegree[task.ID] = len(task.Deps)
	for _, dep := range task.Deps {
		if _, ok := ps.tasks[dep]; !ok {
			ps.tasks[dep] = &Task{ID: dep}
		}
	}
}

// Run executes all tasks using work-stealing with priority ordering.
func (ps *PriorityScheduler) Run() error {
	ps.mu.Lock()
	inDegree := make(map[string]int, len(ps.inDegree))
	for k, v := range ps.inDegree {
		inDegree[k] = v
	}

	tasksByID := make(map[string]*Task, len(ps.tasks))
	for k, v := range ps.tasks {
		tasksByID[k] = v
	}

	type edge struct {
		dep  string
		task string
	}
	var allEdges []edge
	for _, t := range ps.tasks {
		for _, dep := range t.Deps {
			allEdges = append(allEdges, edge{dep: dep, task: t.ID})
		}
	}
	ps.mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(tasksByID))

	readyQueue := make(PriorityQueue, 0)
	for id, deg := range inDegree {
		if deg == 0 && tasksByID[id].Fn != nil {
			heap.Push(&readyQueue, &PQItem{
				task:     tasksByID[id],
				priority: tasksByID[id].Priority,
			})
		}
	}
	heap.Init(&readyQueue)

	var activeWorkers atomic.Int32

	for readyQueue.Len() > 0 {
		item := heap.Pop(&readyQueue).(*PQItem)
		task := item.task

		activeWorkers.Add(1)

		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()
			defer activeWorkers.Add(-1)
			if err := t.Fn(); err != nil {
				t.Error = err
				errCh <- err
			}
		}(task)

		for _, e := range allEdges {
			if e.dep == task.ID {
				inDegree[e.task]--
				if inDegree[e.task] == 0 && tasksByID[e.task].Fn != nil {
					heap.Push(&readyQueue, &PQItem{
						task:     tasksByID[e.task],
						priority: tasksByID[e.task].Priority,
					})
				}
			}
		}
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// TaskCount returns the number of tasks.
func (ps *PriorityScheduler) TaskCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.tasks)
}

// Clear removes all tasks.
func (ps *PriorityScheduler) Clear() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.tasks = make(map[string]*Task)
	ps.inDegree = make(map[string]int)
}

// PQItem is an element in the priority queue.
type PQItem struct {
	task     *Task
	priority int
	index    int
}

// PriorityQueue implements heap.Interface for priority-based scheduling.
type PriorityQueue []*PQItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].priority > pq[j].priority // higher priority first
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*PQItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}
