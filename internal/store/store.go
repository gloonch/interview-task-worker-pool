package store

import (
	"errors"
	"interview-task-worker-pool/internal/domain"
)

var ErrNotFound = errors.New("task not found")

type TaskStore interface {
	Create(t domain.Task) (domain.Task, error)
	Get(id int64) (domain.Task, bool)
	List() ([]domain.Task, error)
}
