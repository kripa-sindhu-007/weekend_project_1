package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/weekend-project/taskqueue/internal/model"
)

type QueuePeekStore struct {
	client *redis.Client
}

func NewQueuePeekStore(client *redis.Client) *QueuePeekStore {
	return &QueuePeekStore{client: client}
}

// PeekReady returns up to limit tasks from the ready queue without removing them.
func (q *QueuePeekStore) PeekReady(ctx context.Context, limit int64) ([]model.Task, error) {
	if limit <= 0 {
		limit = 20
	}
	results, err := q.client.ZRangeWithScores(ctx, KeyReady, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange ready: %w", err)
	}
	tasks := make([]model.Task, 0, len(results))
	for _, z := range results {
		var t model.Task
		if err := json.Unmarshal([]byte(z.Member.(string)), &t); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// PeekDelayed returns up to limit tasks from the delayed queue with their execute_at timestamps.
func (q *QueuePeekStore) PeekDelayed(ctx context.Context, limit int64) ([]model.DelayedEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	results, err := q.client.ZRangeWithScores(ctx, KeyDelayed, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange delayed: %w", err)
	}
	entries := make([]model.DelayedEntry, 0, len(results))
	for _, z := range results {
		var t model.Task
		if err := json.Unmarshal([]byte(z.Member.(string)), &t); err != nil {
			continue
		}
		entries = append(entries, model.DelayedEntry{
			Task:      t,
			ExecuteAt: z.Score,
		})
	}
	return entries, nil
}

// DelayedSize returns the number of tasks in the delayed queue.
func (q *QueuePeekStore) DelayedSize(ctx context.Context) (int64, error) {
	return q.client.ZCard(ctx, KeyDelayed).Result()
}

// DeadLetterSize returns the number of tasks in the dead-letter queue.
func (q *QueuePeekStore) DeadLetterSize(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, KeyDeadLetter).Result()
}
