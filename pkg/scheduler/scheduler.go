package scheduler

import (
	"sync"
)

// Task represents a build task.
type Task struct {
	ID       string
	Name     string
	Deps     []string
	Priority int
	Fn       func() error
	Error    error
}

// Scheduler is a work-stealing task scheduler.
type Scheduler struct {
	tasks    map[string]*Task
	inDegree map[string]int
	mu       sync.Mutex
}

// New creates a new scheduler.
func New() *Scheduler {
	return &Scheduler{
		tasks:    make(map[string]*Task),
		inDegree: make(map[string]int),
	}
}

// AddTask adds a task to the scheduler.
func (s *Scheduler) AddTask(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	s.inDegree[task.ID] = len(task.Deps)
	for _, dep := range task.Deps {
		if _, ok := s.tasks[dep]; !ok {
			s.tasks[dep] = &Task{ID: dep}
		}
	}
}

// Run executes all tasks in dependency order.
func (s *Scheduler) Run() error {
	s.mu.Lock()
	inDegree := make(map[string]int, len(s.inDegree))
	for k, v := range s.inDegree {
		inDegree[k] = v
	}

	tasksByID := make(map[string]*Task, len(s.tasks))
	for k, v := range s.tasks {
		tasksByID[k] = v
	}

	type edge struct {
		dep   string
		task  string
	}
	var allEdges []edge
	for _, t := range s.tasks {
		for _, dep := range t.Deps {
			allEdges = append(allEdges, edge{dep: dep, task: t.ID})
		}
	}
	s.mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(tasksByID))

	var readyQueue []string
	for id, deg := range inDegree {
		if deg == 0 && tasksByID[id].Fn != nil {
			readyQueue = append(readyQueue, id)
		}
	}

	for len(readyQueue) > 0 {
		id := readyQueue[0]
		readyQueue = readyQueue[1:]

		task := tasksByID[id]

		wg.Add(1)
		go func(t *Task) {
			defer wg.Done()
			if err := t.Fn(); err != nil {
				t.Error = err
				errCh <- err
			}
		}(task)

		for _, e := range allEdges {
			if e.dep == id {
				inDegree[e.task]--
				if inDegree[e.task] == 0 && tasksByID[e.task].Fn != nil {
					readyQueue = append(readyQueue, e.task)
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
func (s *Scheduler) TaskCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tasks)
}

// Clear removes all tasks.
func (s *Scheduler) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = make(map[string]*Task)
	s.inDegree = make(map[string]int)
}