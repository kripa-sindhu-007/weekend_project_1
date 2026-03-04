package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/weekend-project/taskqueue/internal/model"
	"github.com/weekend-project/taskqueue/internal/store"
)

type PriorityQueue struct {
	client *redis.Client
}

func NewPriorityQueue(client *redis.Client) *PriorityQueue {
	return &PriorityQueue{client: client}
}

// Enqueue adds a task to the ready queue. Score = -priority so higher priority dequeues first.
func (q *PriorityQueue) Enqueue(ctx context.Context, task model.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return q.client.ZAdd(ctx, store.KeyReady, redis.Z{
		Score:  float64(-task.Priority),
		Member: string(data),
	}).Err()
}

// Dequeue pops the highest-priority task (lowest score) from the ready queue.
func (q *PriorityQueue) Dequeue(ctx context.Context) (*model.Task, error) {
	results, err := q.client.ZPopMin(ctx, store.KeyReady, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("zpopmin: %w", err)
	}
	if len(results) == 0 {
		return nil, nil // queue empty
	}
	var task model.Task
	if err := json.Unmarshal([]byte(results[0].Member.(string)), &task); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}
	return &task, nil
}

// Size returns the number of tasks in the ready queue.
func (q *PriorityQueue) Size(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, store.KeyReady).Result()
}
