package handlers

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/store"
)

// fakeStore is an in-memory DataStore (and services.Datastore) used by handler
// tests. Only the behavior exercised by the tests is meaningful; the rest are
// safe stubs.
type fakeStore struct {
	mu sync.Mutex
	id int64

	usersByEmail map[string]*models.User
	workspace    *models.Workspace
	subscribers  map[string]*models.Subscriber
	notifs       []models.Notification
	deliveries   map[string]*models.Delivery
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		usersByEmail: map[string]*models.User{},
		subscribers:  map[string]*models.Subscriber{},
		deliveries:   map[string]*models.Delivery{},
	}
}

func (f *fakeStore) next(p string) string {
	return p + "_" + itoa(atomic.AddInt64(&f.id, 1))
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// --- users ---

func (f *fakeStore) CreateUser(_ context.Context, email, fullName, passwordHash string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u := &models.User{ID: f.next("user"), Email: email, FullName: fullName, PasswordHash: passwordHash, IsActive: true}
	f.usersByEmail[email] = u
	return u, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.usersByEmail[email]; ok {
		return u, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) GetUserByID(_ context.Context, id string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.usersByEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, store.ErrNotFound
}

// --- workspaces & api keys ---

func (f *fakeStore) CreateWorkspace(_ context.Context, ownerID, name, slug, pub, priv string) (*models.Workspace, error) {
	w := &models.Workspace{ID: f.next("ws"), OwnerID: ownerID, Name: name, Slug: slug, VAPIDPublicKey: pub, VAPIDPrivateKey: priv, NotificationsLimit: 10000}
	f.workspace = w
	return w, nil
}

func (f *fakeStore) GetWorkspaceByID(_ context.Context, id string) (*models.Workspace, error) {
	if f.workspace != nil && f.workspace.ID == id {
		return f.workspace, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) GetWorkspaceByOwner(_ context.Context, _ string) (*models.Workspace, error) {
	if f.workspace != nil {
		return f.workspace, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) UpdateWorkspace(_ context.Context, _, name string) (*models.Workspace, error) {
	if f.workspace == nil {
		return nil, store.ErrNotFound
	}
	f.workspace.Name = name
	return f.workspace, nil
}

func (f *fakeStore) UpdateWorkspaceVAPID(_ context.Context, _, pub, priv string) error {
	if f.workspace == nil {
		return store.ErrNotFound
	}
	f.workspace.VAPIDPublicKey = pub
	f.workspace.VAPIDPrivateKey = priv
	return nil
}

func (f *fakeStore) CreateAPIKey(_ context.Context, workspaceID, name, keyHash, keyPrefix string) (*models.APIKey, error) {
	return &models.APIKey{ID: f.next("key"), WorkspaceID: workspaceID, Name: name, KeyPrefix: keyPrefix, IsActive: true}, nil
}
func (f *fakeStore) ListAPIKeys(_ context.Context, _ string) ([]models.APIKey, error) {
	return nil, nil
}
func (f *fakeStore) DeleteAPIKey(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) ResolveAPIKey(_ context.Context, _ string) (string, error) {
	return "", store.ErrNotFound
}

// --- subscribers ---

func (f *fakeStore) ListSubscribers(_ context.Context, _ string, _, _ int) ([]models.Subscriber, int, error) {
	return nil, 0, nil
}
func (f *fakeStore) GetSubscriber(_ context.Context, workspaceID, externalID string) (*models.Subscriber, error) {
	if s, ok := f.subscribers[workspaceID+"|"+externalID]; ok {
		return s, nil
	}
	return nil, store.ErrNotFound
}
func (f *fakeStore) DeleteSubscriber(_ context.Context, _, _ string) error { return nil }

// --- templates ---

func (f *fakeStore) CreateTemplate(_ context.Context, t *models.Template) (*models.Template, error) {
	t.ID = f.next("tmpl")
	return t, nil
}
func (f *fakeStore) UpdateTemplate(_ context.Context, t *models.Template) (*models.Template, error) {
	return t, nil
}
func (f *fakeStore) GetTemplateByID(_ context.Context, _, _ string) (*models.Template, error) {
	return nil, store.ErrNotFound
}
func (f *fakeStore) GetTemplateBySlug(_ context.Context, _, _ string) (*models.Template, error) {
	return nil, store.ErrNotFound
}
func (f *fakeStore) ListTemplates(_ context.Context, _ string) ([]models.Template, error) {
	return nil, nil
}
func (f *fakeStore) DeleteTemplate(_ context.Context, _, _ string) error { return nil }

// --- notifications & deliveries ---

func (f *fakeStore) CreateNotification(_ context.Context, n *models.Notification) (*models.Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n.ID = f.next("notif")
	f.notifs = append(f.notifs, *n)
	return n, nil
}
func (f *fakeStore) CreateDelivery(_ context.Context, notificationID, workspaceID, channel string, maxAttempts int) (*models.Delivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := &models.Delivery{ID: f.next("del"), NotificationID: notificationID, WorkspaceID: workspaceID, Channel: channel, Status: models.StatusPending, MaxAttempts: maxAttempts}
	f.deliveries[d.ID] = d
	return d, nil
}
func (f *fakeStore) IncrementNotificationsSent(_ context.Context, _ string, _ int64) error {
	return nil
}
func (f *fakeStore) MarkDeliveryResult(_ context.Context, id, status string, attempts int, errMsg *string, _ []byte, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if d, ok := f.deliveries[id]; ok {
		d.Status = status
		d.Attempts = attempts
		d.ErrorMessage = errMsg
	}
	return nil
}
func (f *fakeStore) UpdateNotificationStatus(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) DeactivateWebPush(_ context.Context, _ string) error           { return nil }
func (f *fakeStore) GetNotificationByID(_ context.Context, _, id string) (*models.Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.notifs {
		if f.notifs[i].ID == id {
			return &f.notifs[i], nil
		}
	}
	return nil, store.ErrNotFound
}
func (f *fakeStore) ListNotifications(_ context.Context, _, _ string, _, _ int) ([]models.Notification, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.notifs, len(f.notifs), nil
}
func (f *fakeStore) ListDeliveriesByNotification(_ context.Context, notificationID string) ([]models.Delivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []models.Delivery
	for _, d := range f.deliveries {
		if d.NotificationID == notificationID {
			out = append(out, *d)
		}
	}
	return out, nil
}

// --- webhooks ---

func (f *fakeStore) CreateWebhook(_ context.Context, workspaceID, url, secret string, events []string) (*models.Webhook, error) {
	return &models.Webhook{ID: f.next("wh"), WorkspaceID: workspaceID, URL: url, Secret: secret, Events: events, IsActive: true}, nil
}
func (f *fakeStore) ListWebhooks(_ context.Context, _ string) ([]models.Webhook, error) {
	return nil, nil
}
func (f *fakeStore) DeleteWebhook(_ context.Context, _, _ string) error { return nil }
func (f *fakeStore) ListWebhookLogs(_ context.Context, _, _ string, _ int) ([]models.WebhookLog, error) {
	return nil, nil
}
func (f *fakeStore) ListActiveWebhooksForEvent(_ context.Context, _, _ string) ([]models.Webhook, error) {
	return nil, nil
}
func (f *fakeStore) CreateWebhookLog(_ context.Context, _, _ string, _ []byte, _ *int, _ *string) error {
	return nil
}
