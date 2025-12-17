package service

import (
	"interview-task-worker-pool/internal/domain"
	"strings"
)

type TaskStore interface {
	Create(task domain.Task) (domain.Task, error)
	Get(id int64) (domain.Task, bool)
	List() ([]domain.Task, error)
}

type TaskService struct {
	store TaskStore
}

func New(store TaskStore) (*TaskService, error) {
	if store == nil {
		return nil, ErrStoreNil
	}

	return &TaskService{store: store}, nil
}

func (s *TaskService) CreateTask(title, description string) (domain.Task, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)

	// assumption: description is optional
	if title == "" {
		return domain.Task{}, ErrInvalidInput
	}

	task := domain.Task{
		Title:       title,
		Description: description,
	}

	return s.store.Create(task)
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
