package workerpool

import (
	"context"
	"errors"
	"interview-task-worker-pool/internal/domain"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

var ErrPoolFull = errors.New("task pool is full")
var ErrPoolClosed = errors.New("task pool is closed")

type Store interface {
	Get(id int64) (domain.Task, bool)
	UpdateStatus(id int64, status domain.TaskStatus) (domain.Task, error)
	Fail(id int64, reason string) (domain.Task, error)
}

type TaskPool interface {
	Enqueue(id int64) error
}

type Pool struct {
	mu sync.RWMutex

	queue chan int64
	store Store

	wg        sync.WaitGroup
	closeOnce sync.Once
	closed    atomic.Bool
}

func New(poolSize int, store Store) *Pool {
	return &Pool{
		queue: make(chan int64, poolSize),
		store: store,
	}
}

func (p *Pool) Start(workers int) {
	for i := 0; i < workers; i++ {
		workerID := i + 1
		p.wg.Add(1)
		go p.worker(workerID)
	}
}

func (p *Pool) Enqueue(id int64) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed.Load() {
		return ErrPoolClosed
	}

	select {
	case p.queue <- id:
		return nil
	default:
		return ErrPoolFull
	}
}

func (p *Pool) Shutdown(ctx context.Context) error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed.Store(true)
		close(p.queue)
		p.mu.Unlock()
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) worker(workerID int) {
	defer p.wg.Done()

	for id := range p.queue {
		task, ok := p.store.Get(id)
		if !ok {
			log.Printf("[worker= %d] (taskID= %d) not found.", workerID, id)

			continue
		}
		if task.Status == domain.StatusFailed {
			log.Printf("[worker= %d] (taskID= %d) skipped failed task.", workerID, id)

			continue
		}

		if _, err := p.store.UpdateStatus(id, domain.StatusRunning); err != nil {
			log.Printf("[worker= %d] (taskID= %d) updating status to *RUNNING* failed. (error= %v).", workerID, id, err)

			continue
		}

		// time is measured after the point that task has got RUNNING status
		start := time.Now()

		log.Printf("[worker= %d] (taskID= %d) started task with duration of %s seconds.", workerID, id, task.WorkDuration)

		time.Sleep(task.WorkDuration)

		if _, err := p.store.UpdateStatus(id, domain.StatusDone); err != nil {
			log.Printf("[worker= %d] (taskID= %d) updating status to *DONE* failed. (error= %v).", workerID, id, err)
			continue
		}

		elapsed := time.Since(start)

		log.Printf("[worker= %d] (taskID= %d) completed task with an actual duration of %s (planned %s)", workerID, id, elapsed, task.WorkDuration)
	}
}

func (p *Pool) Queue() <-chan int64 {
	return p.queue
}
