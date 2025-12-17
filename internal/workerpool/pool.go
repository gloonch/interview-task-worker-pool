package workerpool

import (
	"errors"
)

var ErrPoolFull = errors.New("task pool is full")

type TaskPool interface {
	Enqueue(id int64) error
}

type Pool struct {
	queue chan int64
}

func New(poolSize int) *Pool {
	return &Pool{
		queue: make(chan int64, poolSize),
	}
}

func (p *Pool) Enqueue(id int64) error {
	select {
	case p.queue <- id:
		return nil
	default:
		return ErrPoolFull
	}
}

func (p *Pool) Queue() <-chan int64 {
	return p.queue
}
