package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

const deliveryCols = `id, notification_id, workspace_id, channel, status, attempts,
	max_attempts, last_attempt_at, delivered_at, error_message, provider_response,
	created_at, updated_at`

func scanDelivery(row interface{ Scan(...any) error }) (*models.Delivery, error) {
	var d models.Delivery
	err := row.Scan(&d.ID, &d.NotificationID, &d.WorkspaceID, &d.Channel, &d.Status,
		&d.Attempts, &d.MaxAttempts, &d.LastAttemptAt, &d.DeliveredAt, &d.ErrorMessage,
		&d.ProviderResponse, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &d, nil
}

// CreateDelivery inserts a delivery for a notification/channel pair.
func (s *Store) CreateDelivery(ctx context.Context, notificationID, workspaceID, channel string, maxAttempts int) (*models.Delivery, error) {
	return scanDelivery(s.db.QueryRow(ctx, `
		INSERT INTO deliveries (notification_id, workspace_id, channel, status, max_attempts)
		VALUES ($1,$2,$3,'pending',$4)
		RETURNING `+deliveryCols, notificationID, workspaceID, channel, maxAttempts))
}

// GetDeliveryByID fetches a delivery by id.
func (s *Store) GetDeliveryByID(ctx context.Context, id string) (*models.Delivery, error) {
	return scanDelivery(s.db.QueryRow(ctx, `SELECT `+deliveryCols+` FROM deliveries WHERE id=$1`, id))
}

// ListDeliveriesByNotification returns all deliveries for a notification.
func (s *Store) ListDeliveriesByNotification(ctx context.Context, notificationID string) ([]models.Delivery, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+deliveryCols+` FROM deliveries WHERE notification_id=$1 ORDER BY created_at`, notificationID)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()
	var out []models.Delivery
	for rows.Next() {
		d, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// MarkDeliveryResult records the outcome of a delivery attempt.
func (s *Store) MarkDeliveryResult(ctx context.Context, id, status string, attempts int, errMsg *string, providerResp []byte, delivered bool) error {
	if len(providerResp) == 0 {
		providerResp = nil
	}
	_, err := s.db.Exec(ctx, `
		UPDATE deliveries SET
			status = $2,
			attempts = $3,
			error_message = $4,
			provider_response = $5,
			last_attempt_at = NOW(),
			delivered_at = CASE WHEN $6 THEN NOW() ELSE delivered_at END,
			updated_at = NOW()
		WHERE id = $1`, id, status, attempts, errMsg, providerResp, delivered)
	if err != nil {
		return fmt.Errorf("mark delivery result: %w", err)
	}
	return nil
}
