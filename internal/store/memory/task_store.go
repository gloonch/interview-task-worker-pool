package memory

import (
	"errors"
	"interview-task-worker-pool/internal/domain"
	"sync"
	"sync/atomic"
)

var (
	ErrNotInitialized = errors.New("task store not initialized")
)

type TaskStore struct {
	mu     sync.RWMutex
	nextID int64
	tasks  map[int64]domain.Task
}

func New() *TaskStore {
	return &TaskStore{
		tasks: make(map[int64]domain.Task),
	}
}

func (ts *TaskStore) Create(task domain.Task) (domain.Task, error) {
	id := atomic.AddInt64(&ts.nextID, 1)

	task.ID = id

	// status is not definable by user, so here we set its init value
	task.Status = domain.StatusPending

	ts.mu.Lock()
	ts.tasks[id] = task
	ts.mu.Unlock()

	return task, nil
}

func (ts *TaskStore) Get(id int64) (domain.Task, bool) {
	ts.mu.RLock()
	task, ok := ts.tasks[id]
	ts.mu.RUnlock()

	// task is non-pointer value
	return task, ok
}

func (ts *TaskStore) List() ([]domain.Task, error) {

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if ts.tasks == nil {
		return nil, ErrNotInitialized
	}

	tasks := make([]domain.Task, 0, len(ts.tasks))
	for _, t := range ts.tasks {
		tasks = append(tasks, t)
	}

	return tasks, nil
}
