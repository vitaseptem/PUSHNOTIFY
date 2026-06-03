package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisConsumer pulls jobs from the priority queues and dispatches them to
// per-channel handlers, applying retry/backoff and dead-lettering.
type RedisConsumer struct {
	rdb         *redis.Client
	producer    *RedisProducer
	dlq         *DLQ
	log         *zap.Logger
	workers     int
	backoffBase int

	handlers map[string]ChannelHandler
	mu       sync.RWMutex

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewConsumer builds a RedisConsumer.
func NewConsumer(rdb *redis.Client, producer *RedisProducer, log *zap.Logger, workers, backoffBase int) *RedisConsumer {
	if workers < 1 {
		workers = 1
	}
	return &RedisConsumer{
		rdb:         rdb,
		producer:    producer,
		dlq:         NewDLQ(rdb),
		log:         log,
		workers:     workers,
		backoffBase: backoffBase,
		handlers:    make(map[string]ChannelHandler),
	}
}

// SetHandler registers the handler for a channel name.
func (c *RedisConsumer) SetHandler(channel string, handler ChannelHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[channel] = handler
}

// Start launches the worker pool and the delayed-job scheduler. It returns
// immediately; call Stop to drain and shut down.
func (c *RedisConsumer) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	c.wg.Add(1)
	go c.scheduleLoop(ctx)

	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.workerLoop(ctx, i)
	}
	c.log.Info("queue consumer started", zap.Int("workers", c.workers))
}

// Stop signals all goroutines to finish in-flight work and waits for them.
func (c *RedisConsumer) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	c.log.Info("queue consumer stopped")
}

func (c *RedisConsumer) workerLoop(ctx context.Context, id int) {
	defer c.wg.Done()
	queues := []string{KeyQueueHigh, KeyQueueDefault, KeyQueueLow}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// BRPOP blocks across all queues in priority order with a short
		// timeout so we can observe context cancellation promptly.
		res, err := c.rdb.BRPop(ctx, 2*time.Second, queues...).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			c.log.Warn("brpop error", zap.Error(err), zap.Int("worker", id))
			time.Sleep(500 * time.Millisecond)
			continue
		}
		// res[0] is the queue key, res[1] is the payload.
		c.process(ctx, res[1])
	}
}

func (c *RedisConsumer) process(ctx context.Context, raw string) {
	var job QueueJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		c.log.Error("failed to unmarshal job, discarding", zap.Error(err))
		return
	}

	c.mu.RLock()
	handler, ok := c.handlers[job.Channel]
	c.mu.RUnlock()
	if !ok {
		c.log.Error("no handler for channel, dead-lettering",
			zap.String("channel", job.Channel))
		_ = c.dlq.Push(ctx, job)
		return
	}

	job.Attempt++
	err := handler(ctx, job)
	if err == nil {
		return
	}

	c.log.Warn("delivery attempt failed",
		zap.String("delivery", job.DeliveryID),
		zap.String("channel", job.Channel),
		zap.Int("attempt", job.Attempt),
		zap.Error(err))

	if ShouldRetry(job) {
		backoff := CalculateBackoff(job.Attempt, c.backoffBase)
		if err := c.producer.EnqueueDelayed(ctx, job, backoff); err != nil {
			c.log.Error("failed to schedule retry, dead-lettering", zap.Error(err))
			_ = c.dlq.Push(ctx, job)
		}
		return
	}

	c.log.Error("delivery exhausted retries, dead-lettering",
		zap.String("delivery", job.DeliveryID))
	if err := c.dlq.Push(ctx, job); err != nil {
		c.log.Error("failed to push to dlq", zap.Error(err))
	}
}

// scheduleLoop promotes due delayed jobs into their priority queues.
func (c *RedisConsumer) scheduleLoop(ctx context.Context) {
	defer c.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.promoteDue(ctx)
		}
	}
}

func (c *RedisConsumer) promoteDue(ctx context.Context) {
	now := float64(time.Now().Unix())
	// Atomically pop the due members to avoid double-processing across workers.
	res, err := c.rdb.ZRangeByScore(ctx, KeyQueueDelayed, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatFloat(now, 'f', 0, 64),
		Count: 100,
	}).Result()
	if err != nil {
		if ctx.Err() == nil && err != redis.Nil {
			c.log.Warn("zrangebyscore delayed error", zap.Error(err))
		}
		return
	}
	for _, raw := range res {
		// Only the worker that successfully removes the member owns it.
		removed, err := c.rdb.ZRem(ctx, KeyQueueDelayed, raw).Result()
		if err != nil || removed == 0 {
			continue
		}
		var job QueueJob
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			continue
		}
		// Retries go back to the default queue.
		if err := c.rdb.LPush(ctx, KeyQueueDefault, raw).Err(); err != nil {
			c.log.Warn("failed to promote delayed job", zap.Error(err))
		}
	}
}
