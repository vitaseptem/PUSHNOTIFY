package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/channels"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/queue"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"go.uber.org/zap"
)

// ---- in-memory fakes ----

type fakeStore struct {
	mu          sync.Mutex
	ids         int64
	workspaces  map[string]*models.Workspace
	subscribers map[string]*models.Subscriber // key: ws|external
	templates   map[string]*models.Template   // key: ws|slug
	notifs      map[string]*models.Notification
	deliveries  map[string]*models.Delivery
	usageBumped int64
	statusSets  []string
	webPushOff  int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		workspaces:  map[string]*models.Workspace{},
		subscribers: map[string]*models.Subscriber{},
		templates:   map[string]*models.Template{},
		notifs:      map[string]*models.Notification{},
		deliveries:  map[string]*models.Delivery{},
	}
}

func (f *fakeStore) nextID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, atomic.AddInt64(&f.ids, 1))
}

func (f *fakeStore) GetWorkspaceByID(_ context.Context, id string) (*models.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if w, ok := f.workspaces[id]; ok {
		return w, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) GetSubscriber(_ context.Context, workspaceID, externalID string) (*models.Subscriber, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.subscribers[workspaceID+"|"+externalID]; ok {
		return s, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) GetTemplateBySlug(_ context.Context, workspaceID, slug string) (*models.Template, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.templates[workspaceID+"|"+slug]; ok {
		return t, nil
	}
	return nil, store.ErrNotFound
}

func (f *fakeStore) CreateNotification(_ context.Context, n *models.Notification) (*models.Notification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n.ID = f.nextID("notif")
	f.notifs[n.ID] = n
	return n, nil
}

func (f *fakeStore) CreateDelivery(_ context.Context, notificationID, workspaceID, channel string, maxAttempts int) (*models.Delivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := &models.Delivery{
		ID:             f.nextID("del"),
		NotificationID: notificationID,
		WorkspaceID:    workspaceID,
		Channel:        channel,
		Status:         models.StatusPending,
		MaxAttempts:    maxAttempts,
	}
	f.deliveries[d.ID] = d
	return d, nil
}

func (f *fakeStore) IncrementNotificationsSent(_ context.Context, _ string, by int64) error {
	atomic.AddInt64(&f.usageBumped, by)
	return nil
}

func (f *fakeStore) MarkDeliveryResult(_ context.Context, id, status string, attempts int, errMsg *string, _ []byte, delivered bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.deliveries[id]
	if !ok {
		return store.ErrNotFound
	}
	d.Status = status
	d.Attempts = attempts
	d.ErrorMessage = errMsg
	_ = delivered
	return nil
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

func (f *fakeStore) UpdateNotificationStatus(_ context.Context, _, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statusSets = append(f.statusSets, status)
	return nil
}

func (f *fakeStore) DeactivateWebPush(_ context.Context, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.webPushOff++
	return nil
}

func (f *fakeStore) ListActiveWebhooksForEvent(_ context.Context, _, _ string) ([]models.Webhook, error) {
	return nil, nil // no webhooks configured in tests
}

func (f *fakeStore) CreateWebhookLog(_ context.Context, _, _ string, _ []byte, _ *int, _ *string) error {
	return nil
}

func (f *fakeStore) deliveryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.deliveries)
}

// fakeProducer records enqueued jobs.
type fakeProducer struct {
	mu        sync.Mutex
	enqueued  []queue.QueueJob
	delayed   []queue.QueueJob
	bulkCount int
}

func (p *fakeProducer) Enqueue(_ context.Context, job queue.QueueJob, _ int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enqueued = append(p.enqueued, job)
	return nil
}

func (p *fakeProducer) EnqueueDelayed(_ context.Context, job queue.QueueJob, _ time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.delayed = append(p.delayed, job)
	return nil
}

func (p *fakeProducer) EnqueueBulk(_ context.Context, jobs []queue.QueueJob) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.bulkCount += len(jobs)
	return nil
}

func (p *fakeProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.enqueued)
}

// fakeChannel records Send calls and returns a configurable result.
type fakeChannel struct {
	name     string
	result   *channels.DeliveryResult
	err      error
	mu       sync.Mutex
	called   int
	lastBody string
}

func (c *fakeChannel) Name() string    { return c.name }
func (c *fakeChannel) Validate() error { return nil }
func (c *fakeChannel) Send(_ context.Context, job channels.DeliveryJob) (*channels.DeliveryResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.called++
	c.lastBody = job.Body
	return c.result, c.err
}

// ---- helpers ----

func newServiceUnderTest(fs *fakeStore, fp *fakeProducer, chans map[string]channels.Channel) *NotificationService {
	log := zap.NewNop()
	wh := NewWebhookService(fs, log)
	return NewNotificationService(fs, fp, NewTemplateService(), wh, chans, log)
}

func seedWorkspaceAndSubscriber(fs *fakeStore, sent, limit int64) (*models.Workspace, *models.Subscriber) {
	ws := &models.Workspace{ID: "ws_1", NotificationsSent: sent, NotificationsLimit: limit}
	fs.workspaces[ws.ID] = ws
	sub := &models.Subscriber{ID: "sub_1", WorkspaceID: ws.ID, ExternalID: "user_123"}
	fs.subscribers[ws.ID+"|user_123"] = sub
	return ws, sub
}

func okChannels(names ...string) map[string]channels.Channel {
	m := map[string]channels.Channel{}
	for _, n := range names {
		m[n] = &fakeChannel{name: n, result: &channels.DeliveryResult{Success: true}}
	}
	return m
}

// ---- tests ----

func TestSend_LimitReached(t *testing.T) {
	fs := newFakeStore()
	seedWorkspaceAndSubscriber(fs, 100, 100)
	svc := newServiceUnderTest(fs, &fakeProducer{}, okChannels("email"))

	_, err := svc.Send(context.Background(), "ws_1", SendRequest{
		SubscriberID: "user_123", Channels: []string{"email"}, Body: "hi",
	})
	if err == nil || !strings.Contains(err.Error(), "limit reached") {
		t.Fatalf("expected limit reached error, got %v", err)
	}
}

func TestSend_SubscriberNotFound(t *testing.T) {
	fs := newFakeStore()
	fs.workspaces["ws_1"] = &models.Workspace{ID: "ws_1", NotificationsLimit: 100}
	svc := newServiceUnderTest(fs, &fakeProducer{}, okChannels("email"))

	_, err := svc.Send(context.Background(), "ws_1", SendRequest{
		SubscriberID: "ghost", Channels: []string{"email"}, Body: "hi",
	})
	if err == nil {
		t.Fatal("expected error for missing subscriber")
	}
}

func TestSend_CreatesDeliveriesPerChannel(t *testing.T) {
	fs := newFakeStore()
	seedWorkspaceAndSubscriber(fs, 0, 100)
	fp := &fakeProducer{}
	svc := newServiceUnderTest(fs, fp, okChannels("websocket", "email"))

	notif, err := svc.Send(context.Background(), "ws_1", SendRequest{
		SubscriberID: "user_123",
		Channels:     []string{"websocket", "email"},
		Body:         "hello",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(notif.Deliveries) != 2 {
		t.Errorf("expected 2 deliveries on notification, got %d", len(notif.Deliveries))
	}
	if fs.deliveryCount() != 2 {
		t.Errorf("expected 2 deliveries in store, got %d", fs.deliveryCount())
	}
	if fp.count() != 2 {
		t.Errorf("expected 2 enqueued jobs, got %d", fp.count())
	}
}

func TestSend_UsesTemplate(t *testing.T) {
	fs := newFakeStore()
	seedWorkspaceAndSubscriber(fs, 0, 100)
	fs.templates["ws_1|welcome"] = &models.Template{
		WorkspaceID: "ws_1", Slug: "welcome", Channels: []string{"email"},
		Subject: "Hi {{name}}", BodyEmail: "Welcome {{name}}!", Variables: []string{"name"},
	}
	fp := &fakeProducer{}
	svc := newServiceUnderTest(fs, fp, okChannels("email"))

	_, err := svc.Send(context.Background(), "ws_1", SendRequest{
		SubscriberID: "user_123",
		TemplateSlug: "welcome",
		Variables:    map[string]string{"name": "Ana"},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if fp.count() != 1 {
		t.Fatalf("expected 1 job, got %d", fp.count())
	}
	var jp jobPayload
	if err := json.Unmarshal([]byte(fp.enqueued[0].Payload), &jp); err != nil {
		t.Fatalf("unmarshal job payload: %v", err)
	}
	if jp.Body != "Welcome Ana!" {
		t.Errorf("template not rendered, body = %q", jp.Body)
	}
}

func TestSendBulk_ProcessesBatches(t *testing.T) {
	fs := newFakeStore()
	seedWorkspaceAndSubscriber(fs, 0, 1000)
	fp := &fakeProducer{}
	svc := newServiceUnderTest(fs, fp, okChannels("email"))

	reqs := make([]SendRequest, 5)
	for i := range reqs {
		reqs[i] = SendRequest{SubscriberID: "user_123", Channels: []string{"email"}, Body: "hi"}
	}
	if err := svc.SendBulk(context.Background(), "ws_1", reqs); err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if fp.count() != 5 {
		t.Errorf("expected 5 enqueued jobs, got %d", fp.count())
	}
}

func TestHandleDelivery_CallsChannel(t *testing.T) {
	fs := newFakeStore()
	ws, sub := seedWorkspaceAndSubscriber(fs, 0, 100)
	ch := &fakeChannel{name: "email", result: &channels.DeliveryResult{Success: true}}
	svc := newServiceUnderTest(fs, &fakeProducer{}, map[string]channels.Channel{"email": ch})

	del, _ := fs.CreateDelivery(context.Background(), "notif_1", ws.ID, "email", 3)
	payload, _ := json.Marshal(jobPayload{Body: "hi"})
	job := queue.QueueJob{
		DeliveryID: del.ID, NotificationID: "notif_1", WorkspaceID: ws.ID,
		Channel: "email", SubscriberID: sub.ExternalID, Payload: string(payload),
		Attempt: 1, MaxAttempts: 3,
	}
	if err := svc.HandleDelivery(context.Background(), job); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if ch.called != 1 {
		t.Errorf("expected channel called once, got %d", ch.called)
	}
	if fs.deliveries[del.ID].Status != models.StatusDelivered {
		t.Errorf("expected delivered, got %s", fs.deliveries[del.ID].Status)
	}
}

func TestHandleDelivery_MarksFailedOnError(t *testing.T) {
	fs := newFakeStore()
	ws, sub := seedWorkspaceAndSubscriber(fs, 0, 100)
	ch := &fakeChannel{name: "email", err: fmt.Errorf("smtp down")}
	svc := newServiceUnderTest(fs, &fakeProducer{}, map[string]channels.Channel{"email": ch})

	del, _ := fs.CreateDelivery(context.Background(), "notif_1", ws.ID, "email", 3)
	payload, _ := json.Marshal(jobPayload{Body: "hi"})
	job := queue.QueueJob{
		DeliveryID: del.ID, NotificationID: "notif_1", WorkspaceID: ws.ID,
		Channel: "email", SubscriberID: sub.ExternalID, Payload: string(payload),
		Attempt: 1, MaxAttempts: 3,
	}
	// A transport error is returned so the queue can retry.
	if err := svc.HandleDelivery(context.Background(), job); err == nil {
		t.Fatal("expected error to be returned for retry")
	}
	d := fs.deliveries[del.ID]
	if d.Status != models.StatusFailed {
		t.Errorf("expected failed, got %s", d.Status)
	}
	if d.ErrorMessage == nil || !strings.Contains(*d.ErrorMessage, "smtp down") {
		t.Errorf("expected error recorded, got %v", d.ErrorMessage)
	}
}
