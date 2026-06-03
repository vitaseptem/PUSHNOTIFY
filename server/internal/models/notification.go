package models

import (
	"encoding/json"
	"time"
)

// Notification statuses.
const (
	StatusPending   = "pending"
	StatusQueued    = "queued"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusFailed    = "failed"
	StatusPartial   = "partial"
)

// Notification is a single send request that fans out into deliveries.
type Notification struct {
	ID           string          `json:"id"`
	WorkspaceID  string          `json:"workspace_id"`
	SubscriberID *string         `json:"subscriber_id"`
	TemplateID   *string         `json:"template_id"`
	Channels     []string        `json:"channels"`
	Payload      json.RawMessage `json:"payload"`
	Variables    json.RawMessage `json:"variables"`
	Status       string          `json:"status"`
	Priority     int             `json:"priority"`
	ScheduledAt  *time.Time      `json:"scheduled_at"`
	CreatedAt    time.Time       `json:"created_at"`

	// Deliveries is populated when fetching a notification with its children.
	Deliveries []Delivery `json:"deliveries,omitempty"`
}
