package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

const subscriberCols = `id, workspace_id, external_id, email, phone, whatsapp,
	web_push_subscription, metadata, is_active, created_at, updated_at`

func scanSubscriber(row interface{ Scan(...any) error }) (*models.Subscriber, error) {
	var sub models.Subscriber
	err := row.Scan(&sub.ID, &sub.WorkspaceID, &sub.ExternalID, &sub.Email, &sub.Phone,
		&sub.WhatsApp, &sub.WebPushSubscription, &sub.Metadata, &sub.IsActive,
		&sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &sub, nil
}

// UpsertSubscriber creates or updates a subscriber by (workspace, external_id).
func (s *Store) UpsertSubscriber(ctx context.Context, workspaceID, externalID string,
	email, phone, whatsapp *string, metadata []byte) (*models.Subscriber, error) {
	if len(metadata) == 0 {
		metadata = []byte(`{}`)
	}
	return scanSubscriber(s.db.QueryRow(ctx, `
		INSERT INTO subscribers (workspace_id, external_id, email, phone, whatsapp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (workspace_id, external_id) DO UPDATE SET
			email = COALESCE(EXCLUDED.email, subscribers.email),
			phone = COALESCE(EXCLUDED.phone, subscribers.phone),
			whatsapp = COALESCE(EXCLUDED.whatsapp, subscribers.whatsapp),
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING `+subscriberCols, workspaceID, externalID, email, phone, whatsapp, metadata))
}

// GetSubscriber fetches a subscriber by external id within a workspace.
func (s *Store) GetSubscriber(ctx context.Context, workspaceID, externalID string) (*models.Subscriber, error) {
	return scanSubscriber(s.db.QueryRow(ctx,
		`SELECT `+subscriberCols+` FROM subscribers WHERE workspace_id = $1 AND external_id = $2`,
		workspaceID, externalID))
}

// ListSubscribers returns a page of subscribers and the total count.
func (s *Store) ListSubscribers(ctx context.Context, workspaceID string, limit, offset int) ([]models.Subscriber, int, error) {
	var total int
	if err := s.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM subscribers WHERE workspace_id = $1`, workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscribers: %w", err)
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+subscriberCols+` FROM subscribers WHERE workspace_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscribers: %w", err)
	}
	defer rows.Close()
	var out []models.Subscriber
	for rows.Next() {
		sub, err := scanSubscriber(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *sub)
	}
	return out, total, rows.Err()
}

// SetWebPushSubscription stores a subscriber's browser push subscription.
func (s *Store) SetWebPushSubscription(ctx context.Context, workspaceID, externalID string, sub []byte) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE subscribers SET web_push_subscription = $3, updated_at = NOW()
		 WHERE workspace_id = $1 AND external_id = $2`, workspaceID, externalID, sub)
	if err != nil {
		return fmt.Errorf("set web push: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeactivateWebPush clears an expired push subscription.
func (s *Store) DeactivateWebPush(ctx context.Context, subscriberID string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE subscribers SET web_push_subscription = NULL, updated_at = NOW() WHERE id = $1`, subscriberID)
	if err != nil {
		return fmt.Errorf("deactivate web push: %w", err)
	}
	return nil
}

// DeleteSubscriber removes a subscriber by external id.
func (s *Store) DeleteSubscriber(ctx context.Context, workspaceID, externalID string) error {
	tag, err := s.db.Exec(ctx,
		`DELETE FROM subscribers WHERE workspace_id = $1 AND external_id = $2`, workspaceID, externalID)
	if err != nil {
		return fmt.Errorf("delete subscriber: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
