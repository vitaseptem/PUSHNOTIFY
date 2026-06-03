package middleware

import (
	"context"
	"net/http"
)

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxWorkspaceID
)

// WithIdentity stores the authenticated user and workspace on the context.
func WithIdentity(ctx context.Context, userID, workspaceID string) context.Context {
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxWorkspaceID, workspaceID)
	return ctx
}

// WithWorkspace stores just the workspace id (used by API-key auth).
func WithWorkspace(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, ctxWorkspaceID, workspaceID)
}

// UserID returns the authenticated user id, if any.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

// WorkspaceID returns the authenticated workspace id, if any.
func WorkspaceID(ctx context.Context) string {
	v, _ := ctx.Value(ctxWorkspaceID).(string)
	return v
}

// WorkspaceIDFromRequest is a convenience accessor for handlers.
func WorkspaceIDFromRequest(r *http.Request) string {
	return WorkspaceID(r.Context())
}
