package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/channels"
	mw "github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/queue"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"go.uber.org/zap"
)

// noopProducer satisfies queue.Producer for handler tests.
type noopProducer struct{}

func (noopProducer) Enqueue(context.Context, queue.QueueJob, int) error                  { return nil }
func (noopProducer) EnqueueDelayed(context.Context, queue.QueueJob, time.Duration) error { return nil }
func (noopProducer) EnqueueBulk(context.Context, []queue.QueueJob) error                 { return nil }

// okChannel is a channel that always succeeds.
type okChannel struct{ name string }

func (c okChannel) Name() string    { return c.name }
func (c okChannel) Validate() error { return nil }
func (c okChannel) Send(context.Context, channels.DeliveryJob) (*channels.DeliveryResult, error) {
	return &channels.DeliveryResult{Success: true}, nil
}

func newNotifierHandlers(fs *fakeStore) *Handlers {
	log := zap.NewNop()
	chans := map[string]channels.Channel{"email": okChannel{name: "email"}, "websocket": okChannel{name: "websocket"}}
	notifier := services.NewNotificationService(
		fs, noopProducer{}, services.NewTemplateService(), services.NewWebhookService(fs, log), chans, log,
	)
	h := newTestHandlers(fs)
	h.Notifier = notifier
	return h
}

func withWorkspace(req *http.Request, workspaceID string) *http.Request {
	return req.WithContext(mw.WithWorkspace(req.Context(), workspaceID))
}

func seedWS(fs *fakeStore) {
	fs.workspace = &models.Workspace{ID: "ws_1", NotificationsLimit: 10000}
	fs.subscribers["ws_1|user_123"] = &models.Subscriber{ID: "sub_1", WorkspaceID: "ws_1", ExternalID: "user_123"}
}

func TestSendNotification_Success(t *testing.T) {
	fs := newFakeStore()
	seedWS(fs)
	h := newNotifierHandlers(fs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/send",
		strings.NewReader(`{"subscriber_id":"user_123","channels":["email"],"body":"hi"}`))
	req = withWorkspace(req, "ws_1")
	rec := httptest.NewRecorder()
	h.SendNotification(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	var n models.Notification
	if err := json.Unmarshal(rec.Body.Bytes(), &n); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if n.ID == "" || len(n.Deliveries) != 1 {
		t.Errorf("expected notification id and 1 delivery, got %+v", n)
	}
}

func TestSendNotification_MissingBody(t *testing.T) {
	fs := newFakeStore()
	seedWS(fs)
	h := newNotifierHandlers(fs)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/send",
		strings.NewReader(`{"subscriber_id":"user_123","channels":["email"]}`))
	req = withWorkspace(req, "ws_1")
	rec := httptest.NewRecorder()
	h.SendNotification(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "body") {
		t.Errorf("expected a clear message about the body, got %s", rec.Body.String())
	}
}

func TestSendNotification_Unauthorized(t *testing.T) {
	fs := newFakeStore()
	h := newNotifierHandlers(fs)

	// Wrap the handler with the real auth middleware; no API key and no JWT.
	guarded := mw.APIKeyOrJWT(fs, "test-secret")(http.HandlerFunc(h.SendNotification))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/send",
		strings.NewReader(`{"subscriber_id":"user_123","channels":["email"],"body":"hi"}`))
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestSendBulk_Success(t *testing.T) {
	fs := newFakeStore()
	seedWS(fs)
	h := newNotifierHandlers(fs)

	body := `{"notifications":[
		{"subscriber_id":"user_123","channels":["email"],"body":"a"},
		{"subscriber_id":"user_123","channels":["email"],"body":"b"},
		{"subscriber_id":"user_123","channels":["email"],"body":"c"}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/send-bulk", strings.NewReader(body))
	req = withWorkspace(req, "ws_1")
	rec := httptest.NewRecorder()
	h.SendBulkNotifications(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Accepted int `json:"accepted"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Accepted != 3 {
		t.Errorf("expected 3 accepted, got %d", resp.Accepted)
	}
}

func TestListNotifications_Pagination(t *testing.T) {
	fs := newFakeStore()
	seedWS(fs)
	// Seed a couple of notifications.
	for i := 0; i < 2; i++ {
		_, _ = fs.CreateNotification(context.Background(), &models.Notification{WorkspaceID: "ws_1", Channels: []string{"email"}})
	}
	h := newTestHandlers(fs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications?page=1&per_page=10", nil)
	req = withWorkspace(req, "ws_1")
	rec := httptest.NewRecorder()
	h.ListNotifications(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Data    []models.Notification `json:"data"`
		Total   int                   `json:"total"`
		Page    int                   `json:"page"`
		PerPage int                   `json:"per_page"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 2 || resp.Page != 1 || resp.PerPage != 10 || len(resp.Data) != 2 {
		t.Errorf("unexpected pagination: %+v", resp)
	}
}
