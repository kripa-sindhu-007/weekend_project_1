package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/weekend-project/taskqueue/internal/model"
	"github.com/weekend-project/taskqueue/internal/store"
)

type DelayedScheduler struct {
	client *redis.Client
	queue  *PriorityQueue
	events *store.EventStore
}

func NewDelayedScheduler(client *redis.Client, queue *PriorityQueue, events *store.EventStore) *DelayedScheduler {
	return &DelayedScheduler{client: client, queue: queue, events: events}
}

// Schedule adds a task to the delayed set with score = unix timestamp when it should become ready.
func (d *DelayedScheduler) Schedule(ctx context.Context, task model.Task, delay time.Duration) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	executeAt := time.Now().Add(delay).Unix()
	return d.client.ZAdd(ctx, store.KeyDelayed, redis.Z{
		Score:  float64(executeAt),
		Member: string(data),
	}).Err()
}

// Start polls the delayed set every second, moving due tasks to the ready queue.
func (d *DelayedScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Delayed scheduler stopped")
			return
		case <-ticker.C:
			d.promoteDueTasks(ctx)
		}
	}
}

func (d *DelayedScheduler) promoteDueTasks(ctx context.Context) {
	now := float64(time.Now().Unix())
	results, err := d.client.ZRangeByScoreWithScores(ctx, store.KeyDelayed, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: 100,
	}).Result()
	if err != nil {
		log.Printf("Error fetching delayed tasks: %v", err)
		return
	}

	for _, z := range results {
		member := z.Member.(string)
		// Remove from delayed set
		removed, err := d.client.ZRem(ctx, store.KeyDelayed, member).Result()
		if err != nil || removed == 0 {
			continue // another instance grabbed it
		}
		var task model.Task
		if err := json.Unmarshal([]byte(member), &task); err != nil {
			log.Printf("Error unmarshaling delayed task: %v", err)
			continue
		}
		if err := d.queue.Enqueue(ctx, task); err != nil {
			log.Printf("Error promoting delayed task %s: %v", task.ID, err)
		} else {
			log.Printf("Promoted delayed task %s to ready queue", task.ID)
			// Emit promoted event
			if d.events != nil {
				event := model.TaskEvent{
					ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
					TaskID:    task.ID,
					Type:      "promoted",
					WorkerID:  -1,
					Detail:    "Moved from delayed to ready queue",
					Timestamp: time.Now(),
				}
				if err := d.events.Push(ctx, event); err != nil {
					log.Printf("Error pushing promoted event: %v", err)
				}
			}
		}
	}
}
