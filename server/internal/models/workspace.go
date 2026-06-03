package models

import "time"

// Workspace is the multi-tenant boundary. Every resource belongs to one.
type Workspace struct {
	ID                 string    `json:"id"`
	OwnerID            string    `json:"owner_id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Plan               string    `json:"plan"`
	NotificationsSent  int64     `json:"notifications_sent"`
	NotificationsLimit int64     `json:"notifications_limit"`
	VAPIDPublicKey     string    `json:"vapid_public_key"`
	VAPIDPrivateKey    string    `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// APIKey authenticates server-to-server requests for a workspace.
type APIKey struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"`
	KeyPrefix   string     `json:"key_prefix"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
	// PlainKey is only populated once, at creation time, and never stored.
	PlainKey string `json:"key,omitempty"`
}
