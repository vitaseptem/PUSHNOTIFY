package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// CreateAPIKey stores a hashed API key for a workspace.
func (s *Store) CreateAPIKey(ctx context.Context, workspaceID, name, keyHash, keyPrefix string) (*models.APIKey, error) {
	var k models.APIKey
	err := s.db.QueryRow(ctx, `
		INSERT INTO api_keys (workspace_id, name, key_hash, key_prefix)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workspace_id, name, key_prefix, last_used_at, is_active, created_at`,
		workspaceID, name, keyHash, keyPrefix,
	).Scan(&k.ID, &k.WorkspaceID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.IsActive, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", mapErr(err))
	}
	return &k, nil
}

// ListAPIKeys returns all keys for a workspace.
func (s *Store) ListAPIKeys(ctx context.Context, workspaceID string) ([]models.APIKey, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, workspace_id, name, key_prefix, last_used_at, is_active, created_at
		FROM api_keys WHERE workspace_id = $1 ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var keys []models.APIKey
	for rows.Next() {
		var k models.APIKey
		if err := rows.Scan(&k.ID, &k.WorkspaceID, &k.Name, &k.KeyPrefix,
			&k.LastUsedAt, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteAPIKey removes a key scoped to a workspace.
func (s *Store) DeleteAPIKey(ctx context.Context, workspaceID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM api_keys WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveAPIKey returns the workspace id for an active API key hash and
// updates its last-used timestamp.
func (s *Store) ResolveAPIKey(ctx context.Context, keyHash string) (string, error) {
	var workspaceID string
	err := s.db.QueryRow(ctx, `
		UPDATE api_keys SET last_used_at = NOW()
		WHERE key_hash = $1 AND is_active = true
		RETURNING workspace_id`, keyHash,
	).Scan(&workspaceID)
	if err != nil {
		return "", mapErr(err)
	}
	return workspaceID, nil
}
