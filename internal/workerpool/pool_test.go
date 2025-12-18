package workerpool

import (
	"context"
	"errors"
	"interview-task-worker-pool/internal/domain"
	"sync"
	"testing"
	"time"
)

type testStore struct {
	mu      sync.RWMutex
	tasks   map[int64]domain.Task
	running chan int64
	done    chan int64
}

func newTestStore() *testStore {
	return &testStore{
		tasks:   make(map[int64]domain.Task),
		running: make(chan int64, 100),
		done:    make(chan int64, 100),
	}
}

func (ts *testStore) Put(t domain.Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks[t.ID] = t
}

func (ts *testStore) Get(id int64) (domain.Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	t, ok := ts.tasks[id]
	return t, ok
}

func (ts *testStore) UpdateStatus(id int64, status domain.TaskStatus) (domain.Task, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	t, ok := ts.tasks[id]
	if !ok {
		return domain.Task{}, errors.New("task not found")
	}
	t.Status = status
	ts.tasks[id] = t

	switch status {
	case domain.StatusRunning:
		ts.running <- id
	case domain.StatusDone:
		ts.done <- id
	}

	return t, nil
}

func (ts *testStore) Fail(id int64, reason string) (domain.Task, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	t, ok := ts.tasks[id]
	if !ok {
		return domain.Task{}, errors.New("task not found")
	}

	t.Status = domain.StatusFailed
	t.Error = reason
	ts.tasks[id] = t
	return t, nil
}

func waitID(t *testing.T, ch <-chan int64, d time.Duration) int64 {
	t.Helper()

	select {
	case id := <-ch:
		return id
	case <-time.After(d):
		t.Fatalf("timeout waiting for signal %v", d)

		return 0
	}
}

func TestPool_ProcessSingleTask(t *testing.T) {
	store := newTestStore()
	store.Put(domain.Task{
		ID:           1,
		Title:        "t",
		Status:       domain.StatusPending,
		WorkDuration: 20 * time.Millisecond,
	})

	pool := New(10, store)
	pool.Start(1)

	t.Cleanup(func() {
		_ = pool.Shutdown(context.Background())
	})

	if err := pool.Enqueue(1); err != nil {
		t.Fatalf("Enqueue() err=%v, want nil", err)
	}

	// expected get RUNNING then DONE
	gotRunning := waitID(t, store.running, 500*time.Millisecond)
	if gotRunning != 1 {
		t.Fatalf("running id=%d, want 1", gotRunning)
	}

	gotDone := waitID(t, store.done, 1*time.Second)
	if gotDone != 1 {
		t.Fatalf("done id=%d, want 1", gotDone)
	}

	task, ok := store.Get(1)
	if !ok {
		t.Fatalf("Get() ok=false, want true")
	}
	if task.Status != domain.StatusDone {
		t.Fatalf("task.Status=%s, want %s", task.Status, domain.StatusDone)
	}
}

func TestPool_Overflow_ReturnsPoolFull(t *testing.T) {
	store := newTestStore()
	pool := New(1, store)
	pool.Start(0) // with zero workers the queue becomes full

	t.Cleanup(func() {
		_ = pool.Shutdown(context.Background())
	})

	if err := pool.Enqueue(1); err != nil {
		t.Fatalf("first Enqueue() err=%v, want nil", err)
	}
	err := pool.Enqueue(2)
	if err == nil {
		t.Fatalf("second Enqueue() err=nil, want ErrPoolFull")
	}
	if !errors.Is(err, ErrPoolFull) {
		t.Fatalf("second Enqueue() err=%v, want %v", err, ErrPoolFull)
	}
}

func TestPool_Shutdown_DrainsQueuedWork(t *testing.T) {
	store := newTestStore()

	for i := int64(1); i <= 3; i++ {
		store.Put(domain.Task{
			ID:           i,
			Title:        "t",
			Status:       domain.StatusPending,
			WorkDuration: 50 * time.Millisecond,
		})
	}

	pool := New(10, store)
	pool.Start(1)

	if err := pool.Enqueue(1); err != nil {
		t.Fatalf("Enqueue(1) err=%v", err)
	}
	if err := pool.Enqueue(2); err != nil {
		t.Fatalf("Enqueue(2) err=%v", err)
	}
	if err := pool.Enqueue(3); err != nil {
		t.Fatalf("Enqueue(3) err=%v", err)
	}

	// shutdown and wait for workers to get tasks DONE
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- pool.Shutdown(ctx) }()

	// shutdown should not be fast until all tasks are DONE
	select {
	case err := <-errCh:
		t.Fatalf("Shutdown returned too early: %v", err)
	case <-time.After(20 * time.Millisecond):
		// ok: still working
	}

	// wait for all to get DONE
	for i := 0; i < 3; i++ {
		_ = waitID(t, store.done, 1*time.Second)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Shutdown() err=%v, want nil", err)
	}

	for i := int64(1); i <= 3; i++ {
		task, ok := store.Get(i)
		if !ok {
			t.Fatalf("Get(%d) ok=false", i)
		}
		if task.Status != domain.StatusDone {
			t.Fatalf("task %d status=%s, want %s", i, task.Status, domain.StatusDone)
		}
	}
}

func TestPool_EnqueueAfterShutdown_ReturnsPoolClosed(t *testing.T) {
	store := newTestStore()
	pool := New(10, store)
	pool.Start(0)

	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() err=%v, want nil", err)
	}

	err := pool.Enqueue(1)
	if err == nil {
		t.Fatalf("Enqueue() err=nil, want ErrPoolClosed")
	}
	if !errors.Is(err, ErrPoolClosed) {
		t.Fatalf("Enqueue() err=%v, want %v", err, ErrPoolClosed)
	}
}
