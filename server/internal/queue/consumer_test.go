package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func newTestConsumer(t *testing.T) (*RedisConsumer, *RedisProducer, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	producer := NewProducer(rdb)
	// backoffBase 0 makes retries become due immediately, keeping tests fast.
	c := NewConsumer(rdb, producer, zap.NewNop(), 1, 0)
	return c, producer, rdb
}

func TestConsumer_ProcessesJobFromQueue(t *testing.T) {
	c, producer, _ := newTestConsumer(t)
	ctx := context.Background()

	got := make(chan QueueJob, 1)
	c.SetHandler("test", func(_ context.Context, job QueueJob) error {
		got <- job
		return nil
	})

	if err := producer.Enqueue(ctx, QueueJob{DeliveryID: "d1", Channel: "test", MaxAttempts: 3}, 1); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	c.Start(ctx)
	defer c.Stop()

	select {
	case j := <-got:
		if j.DeliveryID != "d1" {
			t.Errorf("wrong job: %+v", j)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("handler not called")
	}
}

func TestConsumer_RetriesOnFailure(t *testing.T) {
	c, producer, rdb := newTestConsumer(t)
	ctx := context.Background()

	var attempts int32
	done := make(chan struct{}, 1)
	c.SetHandler("test", func(_ context.Context, job QueueJob) error {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			return fmt.Errorf("transient failure")
		}
		done <- struct{}{}
		return nil
	})

	if err := producer.Enqueue(ctx, QueueJob{DeliveryID: "d1", Channel: "test", MaxAttempts: 3}, 1); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	c.Start(ctx)
	defer c.Stop()

	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Fatal("job was not retried to success")
	}
	if got := atomic.LoadInt32(&attempts); got < 2 {
		t.Errorf("expected at least 2 attempts, got %d", got)
	}
	if n, _ := rdb.LLen(ctx, KeyQueueDLQ).Result(); n != 0 {
		t.Errorf("job should not be in DLQ, len=%d", n)
	}
}

func TestConsumer_SendsToDLQAfterMaxAttempts(t *testing.T) {
	c, producer, rdb := newTestConsumer(t)
	ctx := context.Background()

	var attempts int32
	c.SetHandler("test", func(_ context.Context, job QueueJob) error {
		atomic.AddInt32(&attempts, 1)
		return fmt.Errorf("always fails")
	})

	if err := producer.Enqueue(ctx, QueueJob{DeliveryID: "d1", Channel: "test", MaxAttempts: 3}, 1); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	c.Start(ctx)
	defer c.Stop()

	// Wait for the job to land in the DLQ after exhausting its 3 attempts.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if n, _ := rdb.LLen(ctx, KeyQueueDLQ).Result(); n == 1 {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}

	if n, _ := rdb.LLen(ctx, KeyQueueDLQ).Result(); n != 1 {
		t.Fatalf("expected 1 job in DLQ, got %d", n)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", got)
	}
	// Nothing should be left in the working queues or the delayed set.
	for _, q := range []string{KeyQueueHigh, KeyQueueDefault, KeyQueueLow} {
		if n, _ := rdb.LLen(ctx, q).Result(); n != 0 {
			t.Errorf("queue %s should be empty, has %d", q, n)
		}
	}
	if n, _ := rdb.ZCard(ctx, KeyQueueDelayed).Result(); n != 0 {
		t.Errorf("delayed set should be empty, has %d", n)
	}
}

func TestConsumer_StopDrainsInFlight(t *testing.T) {
	c, producer, _ := newTestConsumer(t)
	ctx := context.Background()

	var processed int32
	c.SetHandler("test", func(_ context.Context, job QueueJob) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	for i := 0; i < 5; i++ {
		if err := producer.Enqueue(ctx, QueueJob{DeliveryID: fmt.Sprintf("d%d", i), Channel: "test", MaxAttempts: 3}, 5); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}

	c.Start(ctx)
	c.Stop() // must drain all 5 before returning

	if got := atomic.LoadInt32(&processed); got != 5 {
		t.Errorf("expected all 5 jobs drained, processed %d", got)
	}
}
