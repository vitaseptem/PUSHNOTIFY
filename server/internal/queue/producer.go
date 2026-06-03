package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisProducer enqueues jobs using Redis lists (priority queues) and a
// sorted set for delayed delivery.
type RedisProducer struct {
	rdb *redis.Client
}

// NewProducer constructs a RedisProducer.
func NewProducer(rdb *redis.Client) *RedisProducer {
	return &RedisProducer{rdb: rdb}
}

// Enqueue pushes a job onto the priority-appropriate queue.
func (p *RedisProducer) Enqueue(ctx context.Context, job QueueJob, priority int) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	queue := queueForPriority(priority)
	if err := p.rdb.LPush(ctx, queue, data).Err(); err != nil {
		return fmt.Errorf("lpush %s: %w", queue, err)
	}
	return nil
}

// EnqueueDelayed schedules a job to become available after delay.
func (p *RedisProducer) EnqueueDelayed(ctx context.Context, job QueueJob, delay time.Duration) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal delayed job: %w", err)
	}
	runAt := float64(time.Now().Add(delay).Unix())
	if err := p.rdb.ZAdd(ctx, KeyQueueDelayed, redis.Z{Score: runAt, Member: data}).Err(); err != nil {
		return fmt.Errorf("zadd delayed: %w", err)
	}
	return nil
}

// EnqueueBulk enqueues many jobs in a single pipeline, grouped by priority.
func (p *RedisProducer) EnqueueBulk(ctx context.Context, jobs []QueueJob) error {
	if len(jobs) == 0 {
		return nil
	}
	pipe := p.rdb.Pipeline()
	for _, job := range jobs {
		data, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("marshal bulk job: %w", err)
		}
		// Default priority for bulk jobs.
		pipe.LPush(ctx, KeyQueueDefault, data)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("exec bulk pipeline: %w", err)
	}
	return nil
}
