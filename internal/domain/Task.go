package domain

import "time"

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusDone    TaskStatus = "done"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID          int64
	Title       string
	Description string
	Error       string

	Status TaskStatus

	CreatedAt    time.Time
	WorkDuration time.Duration // internal simulation (e.g. 1-5s)
}
