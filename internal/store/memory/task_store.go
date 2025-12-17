package memory

import (
	"errors"
	"interview-task-worker-pool/internal/domain"
	"sync"
	"sync/atomic"
)

var (
	ErrNotFound      = errors.New("task not found")
	ErrInvalidTaskID = errors.New("invalid task id")
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

	tasks := make([]domain.Task, 0, len(ts.tasks))
	for _, t := range ts.tasks {
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (ts *TaskStore) Fail(id int64, reason string) (domain.Task, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	task, ok := ts.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}

	task.Status = domain.StatusFailed
	task.Error = reason
	ts.tasks[id] = task
	return task, nil
}

func (ts *TaskStore) UpdateStatus(id int64, status domain.TaskStatus) (domain.Task, error) {

	ts.mu.Lock()
	defer ts.mu.Unlock()

	task, ok := ts.tasks[id]
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	task.Status = status
	ts.tasks[id] = task

	return task, nil
}
