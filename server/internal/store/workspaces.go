package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// CreateWorkspace inserts a workspace owned by a user.
func (s *Store) CreateWorkspace(ctx context.Context, ownerID, name, slug, vapidPub, vapidPriv string) (*models.Workspace, error) {
	var w models.Workspace
	err := s.db.QueryRow(ctx, `
		INSERT INTO workspaces (owner_id, name, slug, vapid_public_key, vapid_private_key)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, owner_id, name, slug, plan, notifications_sent, notifications_limit,
		          COALESCE(vapid_public_key,''), COALESCE(vapid_private_key,''), created_at, updated_at`,
		ownerID, name, slug, vapidPub, vapidPriv,
	).Scan(&w.ID, &w.OwnerID, &w.Name, &w.Slug, &w.Plan, &w.NotificationsSent,
		&w.NotificationsLimit, &w.VAPIDPublicKey, &w.VAPIDPrivateKey, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create workspace: %w", mapErr(err))
	}
	return &w, nil
}

const workspaceCols = `id, owner_id, name, slug, plan, notifications_sent, notifications_limit,
	COALESCE(vapid_public_key,''), COALESCE(vapid_private_key,''), created_at, updated_at`

func scanWorkspace(row interface {
	Scan(dest ...any) error
}) (*models.Workspace, error) {
	var w models.Workspace
	err := row.Scan(&w.ID, &w.OwnerID, &w.Name, &w.Slug, &w.Plan, &w.NotificationsSent,
		&w.NotificationsLimit, &w.VAPIDPublicKey, &w.VAPIDPrivateKey, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &w, nil
}

// GetWorkspaceByID fetches a workspace by id.
func (s *Store) GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error) {
	return scanWorkspace(s.db.QueryRow(ctx, `SELECT `+workspaceCols+` FROM workspaces WHERE id = $1`, id))
}

// GetWorkspaceByOwner fetches the first workspace owned by a user.
func (s *Store) GetWorkspaceByOwner(ctx context.Context, ownerID string) (*models.Workspace, error) {
	return scanWorkspace(s.db.QueryRow(ctx,
		`SELECT `+workspaceCols+` FROM workspaces WHERE owner_id = $1 ORDER BY created_at LIMIT 1`, ownerID))
}

// UpdateWorkspace updates the mutable workspace fields.
func (s *Store) UpdateWorkspace(ctx context.Context, id, name string) (*models.Workspace, error) {
	return scanWorkspace(s.db.QueryRow(ctx, `
		UPDATE workspaces SET name = $2, updated_at = NOW()
		WHERE id = $1 RETURNING `+workspaceCols, id, name))
}

// UpdateWorkspaceVAPID rotates the workspace VAPID keypair.
func (s *Store) UpdateWorkspaceVAPID(ctx context.Context, id, pub, priv string) error {
	_, err := s.db.Exec(ctx,
		`UPDATE workspaces SET vapid_public_key=$2, vapid_private_key=$3, updated_at=NOW() WHERE id=$1`,
		id, pub, priv)
	if err != nil {
		return fmt.Errorf("update vapid: %w", err)
	}
	return nil
}

// IncrementNotificationsSent atomically bumps the usage counter.
func (s *Store) IncrementNotificationsSent(ctx context.Context, id string, by int64) error {
	_, err := s.db.Exec(ctx,
		`UPDATE workspaces SET notifications_sent = notifications_sent + $2 WHERE id = $1`, id, by)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}
	return nil
}
