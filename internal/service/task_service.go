package service

import (
	"errors"
	"interview-task-worker-pool/internal/domain"
	"interview-task-worker-pool/internal/workerpool"
	"math/rand"
	"strings"
	"time"
)

type TaskStore interface {
	Create(task domain.Task) (domain.Task, error)
	Get(id int64) (domain.Task, bool)
	List() ([]domain.Task, error)
	Fail(id int64, reason string) (domain.Task, error)
}

type TaskService struct {
	store TaskStore
	pool  workerpool.TaskPool
}

func New(store TaskStore, pool workerpool.TaskPool) (*TaskService, error) {
	if store == nil {
		return nil, ErrStoreNil
	}
	if pool == nil {
		return nil, ErrPoolNil
	}

	return &TaskService{store: store, pool: pool}, nil
}

func (s *TaskService) CreateTask(title, description string) (domain.Task, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	// assumption: description is optional
	if title == "" {
		return domain.Task{}, ErrInvalidInput
	}

	task := domain.Task{
		Title:        title,
		Description:  description,
		CreatedAt:    time.Now(),
		WorkDuration: time.Duration(rand.Intn(5)+1) * time.Second,
	}

	created, err := s.store.Create(task)
	if err != nil {
		return domain.Task{}, err
	}

	// Enqueue (non-blocking)
	if err := s.pool.Enqueue(created.ID); err != nil {
		// pool overflow ~> mark task failed and attach reason
		if errors.Is(err, workerpool.ErrPoolFull) {
			failedTask, fErr := s.store.Fail(created.ID, workerpool.ErrPoolFull.Error())
			if fErr != nil {
				return domain.Task{}, fErr
			}
			return failedTask, workerpool.ErrPoolFull
		}
	}
	return created, nil
}

func (s *TaskService) GetTask(id int64) (domain.Task, error) {
	if id <= 0 {
		return domain.Task{}, ErrInvalidID
	}

	task, ok := s.store.Get(id)
	if !ok {
		return domain.Task{}, ErrNotFound
	}
	return task, nil
}

func (s *TaskService) ListTasks() ([]domain.Task, error) {
	return s.store.List()
}
