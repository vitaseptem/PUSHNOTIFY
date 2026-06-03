package handlers

import (
	"context"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// DataStore is the subset of *store.Store used by the HTTP handlers. Depending
// on this interface lets the handlers be unit-tested with in-memory fakes.
// *store.Store satisfies it.
type DataStore interface {
	// Users
	CreateUser(ctx context.Context, email, fullName, passwordHash string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)

	// Workspaces & API keys
	CreateWorkspace(ctx context.Context, ownerID, name, slug, vapidPub, vapidPriv string) (*models.Workspace, error)
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	GetWorkspaceByOwner(ctx context.Context, ownerID string) (*models.Workspace, error)
	UpdateWorkspace(ctx context.Context, id, name string) (*models.Workspace, error)
	UpdateWorkspaceVAPID(ctx context.Context, id, pub, priv string) error
	CreateAPIKey(ctx context.Context, workspaceID, name, keyHash, keyPrefix string) (*models.APIKey, error)
	ListAPIKeys(ctx context.Context, workspaceID string) ([]models.APIKey, error)
	DeleteAPIKey(ctx context.Context, workspaceID, id string) error
	ResolveAPIKey(ctx context.Context, keyHash string) (string, error)

	// Subscribers
	ListSubscribers(ctx context.Context, workspaceID string, limit, offset int) ([]models.Subscriber, int, error)
	GetSubscriber(ctx context.Context, workspaceID, externalID string) (*models.Subscriber, error)
	DeleteSubscriber(ctx context.Context, workspaceID, externalID string) error

	// Templates
	CreateTemplate(ctx context.Context, t *models.Template) (*models.Template, error)
	UpdateTemplate(ctx context.Context, t *models.Template) (*models.Template, error)
	GetTemplateByID(ctx context.Context, workspaceID, id string) (*models.Template, error)
	ListTemplates(ctx context.Context, workspaceID string) ([]models.Template, error)
	DeleteTemplate(ctx context.Context, workspaceID, id string) error

	// Notifications & deliveries
	GetNotificationByID(ctx context.Context, workspaceID, id string) (*models.Notification, error)
	ListNotifications(ctx context.Context, workspaceID, status string, limit, offset int) ([]models.Notification, int, error)
	ListDeliveriesByNotification(ctx context.Context, notificationID string) ([]models.Delivery, error)

	// Webhooks
	CreateWebhook(ctx context.Context, workspaceID, url, secret string, events []string) (*models.Webhook, error)
	ListWebhooks(ctx context.Context, workspaceID string) ([]models.Webhook, error)
	DeleteWebhook(ctx context.Context, workspaceID, id string) error
	ListWebhookLogs(ctx context.Context, workspaceID, webhookID string, limit int) ([]models.WebhookLog, error)
}
