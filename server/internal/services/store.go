package services

import (
	"context"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// Datastore is the subset of *store.Store used by the notification and webhook
// services. Depending on this interface (rather than the concrete store) lets
// those services be unit-tested with in-memory fakes. *store.Store satisfies it.
type Datastore interface {
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetSubscriber(ctx context.Context, workspaceID, externalID string) (*models.Subscriber, error)
	GetTemplateBySlug(ctx context.Context, workspaceID, slug string) (*models.Template, error)
	CreateNotification(ctx context.Context, n *models.Notification) (*models.Notification, error)
	CreateDelivery(ctx context.Context, notificationID, workspaceID, channel string, maxAttempts int) (*models.Delivery, error)
	IncrementNotificationsSent(ctx context.Context, id string, by int64) error
	MarkDeliveryResult(ctx context.Context, id, status string, attempts int, errMsg *string, providerResp []byte, delivered bool) error
	ListDeliveriesByNotification(ctx context.Context, notificationID string) ([]models.Delivery, error)
	UpdateNotificationStatus(ctx context.Context, id, status string) error
	DeactivateWebPush(ctx context.Context, subscriberID string) error
	ListActiveWebhooksForEvent(ctx context.Context, workspaceID, event string) ([]models.Webhook, error)
	CreateWebhookLog(ctx context.Context, webhookID, event string, payload []byte, status *int, body *string) error
}
