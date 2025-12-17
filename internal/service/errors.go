package service

import "errors"

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrStoreNil     = errors.New("task store is nil")
	ErrInvalidID    = errors.New("invalid task id")
)
