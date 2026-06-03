package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

const notificationCols = `id, workspace_id, subscriber_id, template_id, channels,
	payload, variables, status, priority, scheduled_at, created_at`

func scanNotification(row interface{ Scan(...any) error }) (*models.Notification, error) {
	var n models.Notification
	err := row.Scan(&n.ID, &n.WorkspaceID, &n.SubscriberID, &n.TemplateID, &n.Channels,
		&n.Payload, &n.Variables, &n.Status, &n.Priority, &n.ScheduledAt, &n.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &n, nil
}

// CreateNotification inserts a notification row.
func (s *Store) CreateNotification(ctx context.Context, n *models.Notification) (*models.Notification, error) {
	variables := n.Variables
	if len(variables) == 0 {
		variables = []byte(`{}`)
	}
	return scanNotification(s.db.QueryRow(ctx, `
		INSERT INTO notifications (workspace_id, subscriber_id, template_id, channels,
			payload, variables, status, priority, scheduled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+notificationCols,
		n.WorkspaceID, n.SubscriberID, n.TemplateID, n.Channels, n.Payload,
		variables, n.Status, n.Priority, n.ScheduledAt))
}

// UpdateNotificationStatus sets the aggregate status of a notification.
func (s *Store) UpdateNotificationStatus(ctx context.Context, id, status string) error {
	_, err := s.db.Exec(ctx, `UPDATE notifications SET status=$2 WHERE id=$1`, id, status)
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}
	return nil
}

// GetNotificationByID fetches a notification (without deliveries).
func (s *Store) GetNotificationByID(ctx context.Context, workspaceID, id string) (*models.Notification, error) {
	return scanNotification(s.db.QueryRow(ctx,
		`SELECT `+notificationCols+` FROM notifications WHERE id=$1 AND workspace_id=$2`, id, workspaceID))
}

// ListNotifications returns a filtered, paginated list with the total count.
func (s *Store) ListNotifications(ctx context.Context, workspaceID, status string, limit, offset int) ([]models.Notification, int, error) {
	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE workspace_id=$1 AND ($2 = '' OR status = $2)`, workspaceID, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}
	rows, err := s.db.Query(ctx, `
		SELECT `+notificationCols+` FROM notifications
		WHERE workspace_id=$1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC LIMIT $3 OFFSET $4`, workspaceID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	var out []models.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *n)
	}
	return out, total, rows.Err()
}
