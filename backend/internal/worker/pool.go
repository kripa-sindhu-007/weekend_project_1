package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/weekend-project/taskqueue/internal/queue"
	"github.com/weekend-project/taskqueue/internal/store"
)

type Pool struct {
	queue        *queue.PriorityQueue
	executor     *Executor
	workerCount  int
	pollInterval time.Duration
	activeCount  atomic.Int64
	wg           sync.WaitGroup
	workerState  *store.WorkerStateStore
}

func NewPool(
	q *queue.PriorityQueue,
	executor *Executor,
	workerCount int,
	pollIntervalMs int,
	workerState *store.WorkerStateStore,
) *Pool {
	return &Pool{
		queue:        q,
		executor:     executor,
		workerCount:  workerCount,
		pollInterval: time.Duration(pollIntervalMs) * time.Millisecond,
		workerState:  workerState,
	}
}

// Start launches worker goroutines. They poll until ctx is cancelled, then finish in-progress work.
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		// Initialize worker state as idle
		p.workerState.Set(ctx, store.WorkerIdleState(i))
		go p.worker(ctx, i)
	}
	log.Printf("Started %d workers (poll interval: %v)", p.workerCount, p.pollInterval)
}

// Wait blocks until all workers have finished.
func (p *Pool) Wait() {
	p.wg.Wait()
	log.Println("All workers stopped")
}

// ActiveWorkers returns the current number of workers executing a task.
func (p *Pool) ActiveWorkers() int64 {
	return p.activeCount.Load()
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		default:
		}

		task, err := p.queue.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("Worker %d dequeue error: %v", id, err)
			time.Sleep(p.pollInterval)
			continue
		}

		if task == nil {
			// Queue empty, wait before polling again
			time.Sleep(p.pollInterval)
			continue
		}

		p.activeCount.Add(1)
		p.executor.Execute(ctx, task, id)
		p.activeCount.Add(-1)
	}
}
