package models

import (
	"encoding/json"
	"time"
)

// Delivery is the attempt to deliver a notification through one channel.
type Delivery struct {
	ID               string          `json:"id"`
	NotificationID   string          `json:"notification_id"`
	WorkspaceID      string          `json:"workspace_id"`
	Channel          string          `json:"channel"`
	Status           string          `json:"status"`
	Attempts         int             `json:"attempts"`
	MaxAttempts      int             `json:"max_attempts"`
	LastAttemptAt    *time.Time      `json:"last_attempt_at"`
	DeliveredAt      *time.Time      `json:"delivered_at"`
	ErrorMessage     *string         `json:"error_message"`
	ProviderResponse json.RawMessage `json:"provider_response,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// ShouldRetry reports whether the delivery still has attempts remaining.
func (d *Delivery) ShouldRetry() bool {
	return d.Attempts < d.MaxAttempts
}
