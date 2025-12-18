package service

import (
	"errors"
	"testing"

	"interview-task-worker-pool/internal/domain"
	"interview-task-worker-pool/internal/workerpool"
)

// --- fakes ---

type fakeStore struct {
	createFn func(domain.Task) (domain.Task, error)
	getFn    func(int64) (domain.Task, bool)
	listFn   func() ([]domain.Task, error)
	failFn   func(int64, string) (domain.Task, error)
}

func (s *fakeStore) Create(t domain.Task) (domain.Task, error) {
	return s.createFn(t)
}
func (s *fakeStore) Get(id int64) (domain.Task, bool) {
	return s.getFn(id)
}
func (s *fakeStore) List() ([]domain.Task, error) {
	return s.listFn()
}
func (s *fakeStore) Fail(id int64, reason string) (domain.Task, error) {
	return s.failFn(id, reason)
}

type fakePool struct {
	enqueueFn func(int64) error
}

func (p *fakePool) Enqueue(id int64) error {
	return p.enqueueFn(id)
}

// --- tests ---

func TestNew_NilStore(t *testing.T) {
	_, err := New(nil, &fakePool{enqueueFn: func(int64) error { return nil }})
	if err == nil {
		t.Fatalf("New() err=nil, want non-nil")
	}
	if !errors.Is(err, ErrStoreNil) {
		t.Fatalf("New() err=%v, want %v", err, ErrStoreNil)
	}
}

func TestNew_NilPool(t *testing.T) {
	_, err := New(&fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) { return domain.Task{}, nil },
		getFn:    func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn:   func() ([]domain.Task, error) { return nil, nil },
		failFn:   func(int64, string) (domain.Task, error) { return domain.Task{}, nil },
	}, nil)

	if err == nil {
		t.Fatalf("New() err=nil, want non-nil")
	}
	if !errors.Is(err, ErrPoolNil) {
		t.Fatalf("New() err=%v, want %v", err, ErrPoolNil)
	}
}

func TestCreateTask_InvalidInput(t *testing.T) {
	svc, err := New(&fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) {
			t.Fatalf("Create() should not be called on invalid input")
			return domain.Task{}, nil
		},
		getFn:  func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn: func() ([]domain.Task, error) { return nil, nil },
		failFn: func(int64, string) (domain.Task, error) { return domain.Task{}, nil },
	}, &fakePool{enqueueFn: func(int64) error {
		t.Fatalf("Enqueue() should not be called on invalid input")
		return nil
	}})
	if err != nil {
		t.Fatalf("New() err=%v, want nil", err)
	}

	_, e := svc.CreateTask("   ", "desc")
	if e == nil {
		t.Fatalf("CreateTask() err=nil, want ErrInvalidInput")
	}
	if !errors.Is(e, ErrInvalidInput) {
		t.Fatalf("CreateTask() err=%v, want %v", e, ErrInvalidInput)
	}
}

func TestCreateTask_Success_Enqueued(t *testing.T) {
	var createdFromStore domain.Task
	var enqueueCalled bool

	store := &fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) {
			// store should set ID and StatusPending (like memory store)
			createdFromStore = task
			task.ID = 1
			task.Status = domain.StatusPending
			return task, nil
		},
		getFn:  func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn: func() ([]domain.Task, error) { return nil, nil },
		failFn: func(int64, string) (domain.Task, error) { return domain.Task{}, nil },
	}
	pool := &fakePool{
		enqueueFn: func(id int64) error {
			enqueueCalled = true
			if id != 1 {
				t.Fatalf("Enqueue(id)=%d, want 1", id)
			}
			return nil
		},
	}

	svc, err := New(store, pool)
	if err != nil {
		t.Fatalf("New() err=%v, want nil", err)
	}

	out, e := svc.CreateTask("Title", "Desc")
	if e != nil {
		t.Fatalf("CreateTask() err=%v, want nil", e)
	}
	if !enqueueCalled {
		t.Fatalf("Enqueue was not called")
	}
	if out.ID != 1 {
		t.Fatalf("out.ID=%d, want 1", out.ID)
	}
	if out.Status != domain.StatusPending {
		t.Fatalf("out.Status=%s, want %s", out.Status, domain.StatusPending)
	}

	// duration should be set (random 1..5s) – should not be 0
	if createdFromStore.WorkDuration == 0 {
		t.Fatalf("WorkDuration=0, want non-zero")
	}
	if createdFromStore.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt is zero, want non-zero")
	}
}

func TestCreateTask_PoolFull_FailsTask(t *testing.T) {
	store := &fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) {
			task.ID = 10
			task.Status = domain.StatusPending
			return task, nil
		},
		getFn:  func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn: func() ([]domain.Task, error) { return nil, nil },
		failFn: func(id int64, reason string) (domain.Task, error) {
			if id != 10 {
				t.Fatalf("Fail(id)=%d, want 10", id)
			}
			if reason != workerpool.ErrPoolFull.Error() {
				t.Fatalf("Fail(reason)=%q, want %q", reason, workerpool.ErrPoolFull.Error())
			}
			return domain.Task{
				ID:     10,
				Title:  "t",
				Status: domain.StatusFailed,
				Error:  reason,
			}, nil
		},
	}

	pool := &fakePool{
		enqueueFn: func(int64) error { return workerpool.ErrPoolFull },
	}

	svc, _ := New(store, pool)

	task, err := svc.CreateTask("t", "d")
	if err == nil {
		t.Fatalf("CreateTask() err=nil, want %v", workerpool.ErrPoolFull)
	}
	if !errors.Is(err, workerpool.ErrPoolFull) {
		t.Fatalf("CreateTask() err=%v, want %v", err, workerpool.ErrPoolFull)
	}
	if task.Status != domain.StatusFailed {
		t.Fatalf("task.Status=%s, want %s", task.Status, domain.StatusFailed)
	}
	if task.Error != workerpool.ErrPoolFull.Error() {
		t.Fatalf("task.Error=%q, want %q", task.Error, workerpool.ErrPoolFull.Error())
	}
}

func TestCreateTask_PoolClosed_FailsTask(t *testing.T) {
	store := &fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) {
			task.ID = 11
			task.Status = domain.StatusPending
			return task, nil
		},
		getFn:  func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn: func() ([]domain.Task, error) { return nil, nil },
		failFn: func(id int64, reason string) (domain.Task, error) {
			if reason != workerpool.ErrPoolClosed.Error() {
				t.Fatalf("Fail(reason)=%q, want %q", reason, workerpool.ErrPoolClosed.Error())
			}
			return domain.Task{
				ID:     11,
				Title:  "t",
				Status: domain.StatusFailed,
				Error:  reason,
			}, nil
		},
	}
	pool := &fakePool{enqueueFn: func(int64) error { return workerpool.ErrPoolClosed }}

	svc, _ := New(store, pool)

	task, err := svc.CreateTask("t", "d")
	if err == nil {
		t.Fatalf("CreateTask() err=nil, want %v", workerpool.ErrPoolClosed)
	}
	if !errors.Is(err, workerpool.ErrPoolClosed) {
		t.Fatalf("CreateTask() err=%v, want %v", err, workerpool.ErrPoolClosed)
	}
	if task.Status != domain.StatusFailed {
		t.Fatalf("task.Status=%s, want %s", task.Status, domain.StatusFailed)
	}
	if task.Error != workerpool.ErrPoolClosed.Error() {
		t.Fatalf("task.Error=%q, want %q", task.Error, workerpool.ErrPoolClosed.Error())
	}
}

func TestGetTask_InvalidID(t *testing.T) {
	svc, _ := New(&fakeStore{
		createFn: func(task domain.Task) (domain.Task, error) { return domain.Task{}, nil },
		getFn:    func(int64) (domain.Task, bool) { return domain.Task{}, false },
		listFn:   func() ([]domain.Task, error) { return nil, nil },
		failFn:   func(int64, string) (domain.Task, error) { return domain.Task{}, nil },
	}, &fakePool{enqueueFn: func(int64) error { return nil }})

	_, err := svc.GetTask(0)
	if err == nil || !errors.Is(err, ErrInvalidID) {
		t.Fatalf("GetTask() err=%v, want %v", err, ErrInvalidID)
	}
}
