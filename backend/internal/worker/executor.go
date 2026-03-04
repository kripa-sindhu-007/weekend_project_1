package worker

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/weekend-project/taskqueue/internal/model"
	"github.com/weekend-project/taskqueue/internal/queue"
	"github.com/weekend-project/taskqueue/internal/store"
)

type Executor struct {
	delayed     *queue.DelayedScheduler
	deadLetter  *store.DeadLetterStore
	metrics     *store.MetricsStore
	events      *store.EventStore
	workerState *store.WorkerStateStore
}

func NewExecutor(
	delayed *queue.DelayedScheduler,
	deadLetter *store.DeadLetterStore,
	metrics *store.MetricsStore,
	events *store.EventStore,
	workerState *store.WorkerStateStore,
) *Executor {
	return &Executor{
		delayed:     delayed,
		deadLetter:  deadLetter,
		metrics:     metrics,
		events:      events,
		workerState: workerState,
	}
}

// Execute runs the task. ~30% chance of simulated failure.
// On failure: retry with exponential backoff or dead-letter if retries exhausted.
func (e *Executor) Execute(ctx context.Context, task *model.Task, workerID int) {
	log.Printf("Executing task %s (priority=%d, attempt=%d/%d)",
		task.ID, task.Priority, task.Retries+1, task.MaxRetries+1)

	// Set worker state to processing
	e.workerState.Set(ctx, model.WorkerState{
		ID:        workerID,
		Status:    "processing",
		TaskID:    task.ID,
		StartedAt: time.Now(),
	})

	// Emit started event
	e.emitEvent(ctx, task.ID, "started", workerID, fmt.Sprintf("Worker %d picked up task", workerID))

	// Simulate work (200-800ms)
	workDuration := time.Duration(200+rand.Intn(600)) * time.Millisecond
	time.Sleep(workDuration)

	// ~30% simulated failure rate
	if rand.Float64() < 0.3 {
		errMsg := fmt.Sprintf("simulated failure for task %s", task.ID)
		log.Printf("Task %s failed: %s", task.ID, errMsg)
		task.Error = errMsg
		e.emitEvent(ctx, task.ID, "failed", workerID, errMsg)
		e.handleFailure(ctx, task, workerID)
	} else {
		// Success
		task.Status = model.StatusCompleted
		log.Printf("Task %s completed successfully", task.ID)
		e.emitEvent(ctx, task.ID, "completed", workerID, fmt.Sprintf("Completed in %v", workDuration))
		if err := e.metrics.IncrProcessed(ctx); err != nil {
			log.Printf("Error incrementing processed metric: %v", err)
		}
	}

	// Set worker state back to idle
	e.workerState.Set(ctx, model.WorkerState{
		ID:     workerID,
		Status: "idle",
	})
}

func (e *Executor) handleFailure(ctx context.Context, task *model.Task, workerID int) {
	if task.Retries < task.MaxRetries {
		// Retry with exponential backoff: min(2^retries * 1s, 60s)
		task.Retries++
		task.Status = model.StatusPending
		backoff := math.Min(math.Pow(2, float64(task.Retries)), 60)
		delay := time.Duration(backoff) * time.Second

		log.Printf("Retrying task %s in %v (attempt %d/%d)",
			task.ID, delay, task.Retries+1, task.MaxRetries+1)

		e.emitEvent(ctx, task.ID, "retrying", workerID,
			fmt.Sprintf("Retry %d/%d in %v", task.Retries, task.MaxRetries, delay))

		if err := e.metrics.IncrRetries(ctx); err != nil {
			log.Printf("Error incrementing retries metric: %v", err)
		}
		if err := e.delayed.Schedule(ctx, *task, delay); err != nil {
			log.Printf("Error scheduling retry for task %s: %v", task.ID, err)
		}
	} else {
		// Exhausted retries — dead-letter
		task.Status = model.StatusFailed
		log.Printf("Task %s exhausted retries, moving to dead-letter", task.ID)

		e.emitEvent(ctx, task.ID, "dead_lettered", workerID,
			fmt.Sprintf("Exhausted %d retries", task.MaxRetries))

		ft := model.FailedTask{
			Task:     *task,
			FailedAt: time.Now(),
			Reason:   task.Error,
		}
		if err := e.deadLetter.Push(ctx, ft); err != nil {
			log.Printf("Error pushing to dead-letter: %v", err)
		}
		if err := e.metrics.IncrFailed(ctx); err != nil {
			log.Printf("Error incrementing failed metric: %v", err)
		}
	}
}

func (e *Executor) emitEvent(ctx context.Context, taskID, eventType string, workerID int, detail string) {
	event := model.TaskEvent{
		ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		TaskID:    taskID,
		Type:      eventType,
		WorkerID:  workerID,
		Detail:    detail,
		Timestamp: time.Now(),
	}
	if err := e.events.Push(ctx, event); err != nil {
		log.Printf("Error pushing event: %v", err)
	}
}
