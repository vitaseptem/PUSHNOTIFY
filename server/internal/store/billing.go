package store

import (
	"context"
	"fmt"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// SetWorkspaceStripeCustomer stores the Stripe customer id for a workspace.
func (s *Store) SetWorkspaceStripeCustomer(ctx context.Context, workspaceID, customerID string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE workspaces SET stripe_customer_id = $2, updated_at = NOW() WHERE id = $1`,
		workspaceID, customerID)
	if err != nil {
		return fmt.Errorf("set stripe customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetWorkspaceByStripeCustomer looks up a workspace by its Stripe customer id.
func (s *Store) GetWorkspaceByStripeCustomer(ctx context.Context, customerID string) (*models.Workspace, error) {
	return scanWorkspace(s.db.QueryRow(ctx,
		`SELECT `+workspaceCols+` FROM workspaces WHERE stripe_customer_id = $1`, customerID))
}

// UpdateWorkspaceSubscription updates a workspace's plan, limit, subscription
// status and billing period in one statement (called from the Stripe webhook).
func (s *Store) UpdateWorkspaceSubscription(ctx context.Context, workspaceID, plan string, limit int64, status string, subscriptionID *string, periodEnd *time.Time) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE workspaces SET
			plan = $2,
			notifications_limit = $3,
			plan_status = $4,
			stripe_subscription_id = $5,
			current_period_end = $6,
			updated_at = NOW()
		WHERE id = $1`,
		workspaceID, plan, limit, status, subscriptionID, periodEnd)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
