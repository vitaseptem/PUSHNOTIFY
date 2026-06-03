package models

import (
	"encoding/json"
	"time"
)

// Subscriber is an end user that can receive notifications across channels.
type Subscriber struct {
	ID                  string          `json:"id"`
	WorkspaceID         string          `json:"workspace_id"`
	ExternalID          string          `json:"external_id"`
	Email               *string         `json:"email"`
	Phone               *string         `json:"phone"`
	WhatsApp            *string         `json:"whatsapp"`
	WebPushSubscription json.RawMessage `json:"web_push_subscription,omitempty"`
	Metadata            json.RawMessage `json:"metadata"`
	IsActive            bool            `json:"is_active"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}
