package queue

import (
	"context"
	"time"
)

// Redis key names for the queue subsystem.
const (
	KeyQueueHigh    = "pushnotify:queue:high"
	KeyQueueDefault = "pushnotify:queue:default"
	KeyQueueLow     = "pushnotify:queue:low"
	KeyQueueDelayed = "pushnotify:queue:delayed" // sorted set scored by run-at unix
	KeyQueueDLQ     = "pushnotify:queue:dlq"
)

// QueueJob is a single unit of work: deliver one notification on one channel.
type QueueJob struct {
	DeliveryID     string    `json:"delivery_id"`
	NotificationID string    `json:"notification_id"`
	WorkspaceID    string    `json:"workspace_id"`
	Channel        string    `json:"channel"`
	SubscriberID   string    `json:"subscriber_id"`
	Payload        string    `json:"payload"`
	Attempt        int       `json:"attempt"`
	MaxAttempts    int       `json:"max_attempts"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChannelHandler processes a job for a specific channel. Returning an error
// signals a failed attempt, which the consumer will retry or dead-letter.
type ChannelHandler func(ctx context.Context, job QueueJob) error

// Producer enqueues jobs onto the Redis-backed queues.
type Producer interface {
	Enqueue(ctx context.Context, job QueueJob, priority int) error
	EnqueueDelayed(ctx context.Context, job QueueJob, delay time.Duration) error
	EnqueueBulk(ctx context.Context, jobs []QueueJob) error
}

// Consumer pulls jobs off the queues and dispatches them to channel handlers.
type Consumer interface {
	Start(ctx context.Context)
	Stop()
	SetHandler(channel string, handler ChannelHandler)
}

// queueForPriority maps a 1-10 priority into one of the three queues.
func queueForPriority(priority int) string {
	switch {
	case priority <= 3:
		return KeyQueueHigh
	case priority <= 7:
		return KeyQueueDefault
	default:
		return KeyQueueLow
	}
}
