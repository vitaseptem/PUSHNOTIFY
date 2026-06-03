package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// CreateWebhook stores a webhook endpoint for a workspace.
func (s *Store) CreateWebhook(ctx context.Context, workspaceID, url, secret string, events []string) (*models.Webhook, error) {
	var w models.Webhook
	err := s.db.QueryRow(ctx, `
		INSERT INTO webhooks (workspace_id, url, secret, events)
		VALUES ($1,$2,$3,$4)
		RETURNING id, workspace_id, url, secret, events, is_active, created_at`,
		workspaceID, url, secret, events,
	).Scan(&w.ID, &w.WorkspaceID, &w.URL, &w.Secret, &w.Events, &w.IsActive, &w.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create webhook: %w", mapErr(err))
	}
	return &w, nil
}

// ListWebhooks returns a workspace's webhooks.
func (s *Store) ListWebhooks(ctx context.Context, workspaceID string) ([]models.Webhook, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, workspace_id, url, secret, events, is_active, created_at
		FROM webhooks WHERE workspace_id=$1 ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()
	var out []models.Webhook
	for rows.Next() {
		var w models.Webhook
		if err := rows.Scan(&w.ID, &w.WorkspaceID, &w.URL, &w.Secret, &w.Events,
			&w.IsActive, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ListActiveWebhooksForEvent returns active webhooks subscribed to an event.
func (s *Store) ListActiveWebhooksForEvent(ctx context.Context, workspaceID, event string) ([]models.Webhook, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, workspace_id, url, secret, events, is_active, created_at
		FROM webhooks WHERE workspace_id=$1 AND is_active=true AND $2 = ANY(events)`,
		workspaceID, event)
	if err != nil {
		return nil, fmt.Errorf("list active webhooks: %w", err)
	}
	defer rows.Close()
	var out []models.Webhook
	for rows.Next() {
		var w models.Webhook
		if err := rows.Scan(&w.ID, &w.WorkspaceID, &w.URL, &w.Secret, &w.Events,
			&w.IsActive, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// DeleteWebhook removes a webhook from a workspace.
func (s *Store) DeleteWebhook(ctx context.Context, workspaceID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM webhooks WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateWebhookLog records a webhook dispatch attempt.
func (s *Store) CreateWebhookLog(ctx context.Context, webhookID, event string, payload []byte, status *int, body *string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO webhook_logs (webhook_id, event, payload, response_status, response_body, delivered_at)
		VALUES ($1,$2,$3,$4,$5, CASE WHEN $4 IS NOT NULL THEN NOW() ELSE NULL END)`,
		webhookID, event, payload, status, body)
	if err != nil {
		return fmt.Errorf("create webhook log: %w", err)
	}
	return nil
}

// ListWebhookLogs returns recent logs for a webhook.
func (s *Store) ListWebhookLogs(ctx context.Context, workspaceID, webhookID string, limit int) ([]models.WebhookLog, error) {
	rows, err := s.db.Query(ctx, `
		SELECT l.id, l.webhook_id, l.event, l.payload, l.response_status, l.response_body,
		       l.delivered_at, l.created_at
		FROM webhook_logs l
		JOIN webhooks w ON w.id = l.webhook_id
		WHERE l.webhook_id=$1 AND w.workspace_id=$2
		ORDER BY l.created_at DESC LIMIT $3`, webhookID, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("list webhook logs: %w", err)
	}
	defer rows.Close()
	var out []models.WebhookLog
	for rows.Next() {
		var l models.WebhookLog
		if err := rows.Scan(&l.ID, &l.WebhookID, &l.Event, &l.Payload, &l.ResponseStatus,
			&l.ResponseBody, &l.DeliveredAt, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
