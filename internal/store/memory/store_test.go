package memory

import (
	"errors"
	"interview-task-worker-pool/internal/domain"
	"sync"
	"testing"
)

func TestTaskStore_CreateAndGet(t *testing.T) {
	ts := New()

	in := domain.Task{
		Title:       "t1",
		Description: "d1",
		Status:      domain.StatusDone,
	}

	created, err := ts.Create(in)
	if err != nil {
		t.Fatalf("Create() err = %v, want nil", err)
	}
	if created.ID <= 0 {
		t.Fatalf("Create() err = %d, want > 0", created.ID)
	}
	if created.Status != domain.StatusPending {
		t.Fatalf("Create() err = %s, want %s", created.Status, domain.StatusPending)
	}

	got, ok := ts.Get(created.ID)
	if !ok {
		t.Fatal("Get() ok = false, want ok = true")
	}
	if got.ID != created.ID || got.Title != in.Title || got.Description != in.Description {
		t.Fatalf("Get() returned unxepected task: %+v", got)
	}
	if got.Status != domain.StatusPending {
		t.Fatalf("Get() status = %s, want %s", got.Status, domain.StatusPending)
	}
}

func TestTaskStore_Get_NotFound(t *testing.T) {
	ts := New()

	_, ok := ts.Get(9999)
	if ok {
		t.Fatal("Get() ok = true, want ok = false")
	}
}

func TestTaskStore_List(t *testing.T) {
	ts := New()

	t1, _ := ts.Create(domain.Task{Title: "t1"})
	t2, _ := ts.Create(domain.Task{Title: "t2"})

	list, err := ts.List()
	if err != nil {
		t.Fatalf("List() err = %v, want nil", err)
	}
	if len(list) != 2 {
		t.Fatalf("List() len = %d, want 2", len(list))
	}

	// not sorted because of map
	if !containsID(list, t1.ID) || !containsID(list, t2.ID) {
		t.Fatalf("List() does not contain created tasks in a list with length of %v", len(list))
	}
}

func TestTaskStore_Fail(t *testing.T) {
	ts := New()

	created, _ := ts.Create(domain.Task{Title: "t"})
	reason := "task pool is full"

	failed, err := ts.Fail(created.ID, reason)
	if err != nil {
		t.Fatalf("Fail() err = %v, want nil", err)
	}
	if failed.Status != domain.StatusFailed {
		t.Fatalf("Fail() status = %s, want %s", failed.Status, domain.StatusFailed)
	}
	if failed.Error != reason {
		t.Fatalf("Fail() error = %s, want %s", failed.Error, reason)
	}

	got, ok := ts.Get(created.ID)
	if !ok {
		t.Fatalf("Get() ok = false, want ok = true")
	}
	if got.Status != domain.StatusFailed || got.Error != reason {
		t.Fatalf("Get() after Fail got= %+v, want status=failed error=%q", got, reason)
	}
}

func TestTaskStore_Fail_NotFound(t *testing.T) {
	ts := New()

	_, err := ts.Fail(123, "x")
	if err == nil {
		t.Fatalf("Fail() err = nil, want non-nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Fail() err = %v, want %v", err, ErrNotFound)
	}
}

func TestTaskStore_UpdateStatus(t *testing.T) {
	ts := New()

	created, _ := ts.Create(domain.Task{Title: "t"})
	updated, err := ts.UpdateStatus(created.ID, domain.StatusRunning)
	if err != nil {
		t.Fatalf("UpdateStatus() err = %v, want nil", err)
	}
	if updated.Status != domain.StatusRunning {
		t.Fatalf("UpdateStatus() Status = %s, want %s", updated.Status, domain.StatusRunning)
	}

	got, ok := ts.Get(created.ID)
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if got.Status != domain.StatusRunning {
		t.Fatalf("Get() Status = %s, want %s", got.Status, domain.StatusRunning)
	}
}

func TestTaskStore_UpdateStatus_NotFound(t *testing.T) {
	ts := New()

	_, err := ts.UpdateStatus(321, domain.StatusDone)
	if err == nil {
		t.Fatalf("UpdateStatus() err = nil, want non-nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateStatus() err = %v, want %v", err, ErrNotFound)
	}
}

func TestTaskStore_ConcurrentCreate(t *testing.T) {
	ts := New()

	const n = 200
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = ts.Create(domain.Task{Title: "x"})
		}()
	}

	wg.Wait()

	list, err := ts.List()
	if err != nil {
		t.Fatalf("List() err = %v, want nil", err)
	}
	if len(list) != n {
		t.Fatalf("List() len = %d, want %d", len(list), n)
	}
}

func containsID(tasks []domain.Task, id int64) bool {
	for _, t := range tasks {
		if t.ID == id {
			return true
		}
	}
	return false
}
