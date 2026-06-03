package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/store"
)

// SubscriberService manages subscriber records and their push subscriptions.
type SubscriberService struct {
	store *store.Store
}

// NewSubscriberService constructs the service.
func NewSubscriberService(st *store.Store) *SubscriberService {
	return &SubscriberService{store: st}
}

// SubscriberInput is the upsert payload.
type SubscriberInput struct {
	ExternalID string                 `json:"external_id"`
	Email      *string                `json:"email"`
	Phone      *string                `json:"phone"`
	WhatsApp   *string                `json:"whatsapp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Upsert creates or updates a subscriber.
func (s *SubscriberService) Upsert(ctx context.Context, workspaceID string, in SubscriberInput) (*models.Subscriber, error) {
	if in.ExternalID == "" {
		return nil, fmt.Errorf("external_id is required")
	}
	var meta []byte
	if in.Metadata != nil {
		meta, _ = json.Marshal(in.Metadata)
	}
	return s.store.UpsertSubscriber(ctx, workspaceID, in.ExternalID, in.Email, in.Phone, in.WhatsApp, meta)
}

// WebPushSubscription is the browser-provided PushSubscription shape.
type WebPushSubscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

// RegisterWebPush validates and stores a browser push subscription.
func (s *SubscriberService) RegisterWebPush(ctx context.Context, workspaceID, externalID string, sub WebPushSubscription) error {
	if sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		return fmt.Errorf("invalid web push subscription: endpoint and keys are required")
	}
	raw, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("marshal subscription: %w", err)
	}
	return s.store.SetWebPushSubscription(ctx, workspaceID, externalID, raw)
}
