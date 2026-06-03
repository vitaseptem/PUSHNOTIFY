package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// DLQ wraps dead-letter-queue operations.
type DLQ struct {
	rdb *redis.Client
}

// NewDLQ constructs a DLQ helper.
func NewDLQ(rdb *redis.Client) *DLQ { return &DLQ{rdb: rdb} }

// Push moves a permanently failed job into the dead letter queue.
func (d *DLQ) Push(ctx context.Context, job QueueJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal dlq job: %w", err)
	}
	if err := d.rdb.LPush(ctx, KeyQueueDLQ, data).Err(); err != nil {
		return fmt.Errorf("lpush dlq: %w", err)
	}
	return nil
}

// Len returns the number of jobs currently in the DLQ.
func (d *DLQ) Len(ctx context.Context) (int64, error) {
	n, err := d.rdb.LLen(ctx, KeyQueueDLQ).Result()
	if err != nil {
		return 0, fmt.Errorf("llen dlq: %w", err)
	}
	return n, nil
}

// Peek returns up to limit jobs from the DLQ without removing them.
func (d *DLQ) Peek(ctx context.Context, limit int64) ([]QueueJob, error) {
	raws, err := d.rdb.LRange(ctx, KeyQueueDLQ, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange dlq: %w", err)
	}
	jobs := make([]QueueJob, 0, len(raws))
	for _, raw := range raws {
		var job QueueJob
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
