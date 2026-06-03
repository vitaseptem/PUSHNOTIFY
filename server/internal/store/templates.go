package store

import (
	"context"
	"fmt"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

const templateCols = `id, workspace_id, name, slug, channels,
	COALESCE(subject,''), COALESCE(body_websocket,''), COALESCE(body_webpush,''),
	COALESCE(body_email,''), COALESCE(body_whatsapp,''), COALESCE(body_sms,''),
	variables, created_at, updated_at`

func scanTemplate(row interface{ Scan(...any) error }) (*models.Template, error) {
	var t models.Template
	err := row.Scan(&t.ID, &t.WorkspaceID, &t.Name, &t.Slug, &t.Channels, &t.Subject,
		&t.BodyWebSocket, &t.BodyWebPush, &t.BodyEmail, &t.BodyWhatsApp, &t.BodySMS,
		&t.Variables, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &t, nil
}

// CreateTemplate inserts a template.
func (s *Store) CreateTemplate(ctx context.Context, t *models.Template) (*models.Template, error) {
	return scanTemplate(s.db.QueryRow(ctx, `
		INSERT INTO templates (workspace_id, name, slug, channels, subject,
			body_websocket, body_webpush, body_email, body_whatsapp, body_sms, variables)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+templateCols,
		t.WorkspaceID, t.Name, t.Slug, t.Channels, t.Subject,
		t.BodyWebSocket, t.BodyWebPush, t.BodyEmail, t.BodyWhatsApp, t.BodySMS, t.Variables))
}

// UpdateTemplate updates a template by id within a workspace.
func (s *Store) UpdateTemplate(ctx context.Context, t *models.Template) (*models.Template, error) {
	return scanTemplate(s.db.QueryRow(ctx, `
		UPDATE templates SET name=$3, slug=$4, channels=$5, subject=$6,
			body_websocket=$7, body_webpush=$8, body_email=$9, body_whatsapp=$10,
			body_sms=$11, variables=$12, updated_at=NOW()
		WHERE id=$1 AND workspace_id=$2
		RETURNING `+templateCols,
		t.ID, t.WorkspaceID, t.Name, t.Slug, t.Channels, t.Subject,
		t.BodyWebSocket, t.BodyWebPush, t.BodyEmail, t.BodyWhatsApp, t.BodySMS, t.Variables))
}

// GetTemplateByID fetches a template by id within a workspace.
func (s *Store) GetTemplateByID(ctx context.Context, workspaceID, id string) (*models.Template, error) {
	return scanTemplate(s.db.QueryRow(ctx,
		`SELECT `+templateCols+` FROM templates WHERE id=$1 AND workspace_id=$2`, id, workspaceID))
}

// GetTemplateBySlug fetches a template by slug within a workspace.
func (s *Store) GetTemplateBySlug(ctx context.Context, workspaceID, slug string) (*models.Template, error) {
	return scanTemplate(s.db.QueryRow(ctx,
		`SELECT `+templateCols+` FROM templates WHERE slug=$1 AND workspace_id=$2`, slug, workspaceID))
}

// ListTemplates returns all templates for a workspace.
func (s *Store) ListTemplates(ctx context.Context, workspaceID string) ([]models.Template, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+templateCols+` FROM templates WHERE workspace_id=$1 ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()
	var out []models.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// DeleteTemplate removes a template by id within a workspace.
func (s *Store) DeleteTemplate(ctx context.Context, workspaceID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM templates WHERE id=$1 AND workspace_id=$2`, id, workspaceID)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
