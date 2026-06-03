package models

import (
	"encoding/json"
	"time"
)

// Webhook is a customer-configured HTTP endpoint for delivery status events.
type Webhook struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	URL         string    `json:"url"`
	Secret      string    `json:"-"`
	Events      []string  `json:"events"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// WebhookLog records a single webhook dispatch attempt.
type WebhookLog struct {
	ID             string          `json:"id"`
	WebhookID      string          `json:"webhook_id"`
	Event          string          `json:"event"`
	Payload        json.RawMessage `json:"payload"`
	ResponseStatus *int            `json:"response_status"`
	ResponseBody   *string         `json:"response_body"`
	DeliveredAt    *time.Time      `json:"delivered_at"`
	CreatedAt      time.Time       `json:"created_at"`
}
