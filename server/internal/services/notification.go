package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/channels"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	"github.com/astrazstudio/pushnotify/server/internal/queue"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"go.uber.org/zap"
)

// SendRequest is a single notification send instruction.
type SendRequest struct {
	SubscriberID string                 `json:"subscriber_id"`
	Channels     []string               `json:"channels"`
	Subject      string                 `json:"subject"`
	Body         string                 `json:"body"`
	TemplateSlug string                 `json:"template"`
	Variables    map[string]string      `json:"variables"`
	Metadata     map[string]interface{} `json:"metadata"`
	Priority     int                    `json:"priority"`
	ScheduledAt  *time.Time             `json:"scheduled_at"`
}

// jobPayload is the per-delivery content carried on the queue.
type jobPayload struct {
	Subject  string                 `json:"subject"`
	Body     string                 `json:"body"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NotificationService orchestrates send requests into queued deliveries and
// processes those deliveries through channels when run as a worker.
type NotificationService struct {
	store    *store.Store
	producer queue.Producer
	tmpl     *TemplateService
	webhooks *WebhookService
	channels map[string]channels.Channel
	log      *zap.Logger
}

// NewNotificationService wires the orchestrator.
func NewNotificationService(
	st *store.Store,
	producer queue.Producer,
	tmpl *TemplateService,
	webhooks *WebhookService,
	chans map[string]channels.Channel,
	log *zap.Logger,
) *NotificationService {
	return &NotificationService{
		store:    st,
		producer: producer,
		tmpl:     tmpl,
		webhooks: webhooks,
		channels: chans,
		log:      log,
	}
}

// Send validates the request, persists the notification and its deliveries,
// and enqueues a job per channel.
func (s *NotificationService) Send(ctx context.Context, workspaceID string, req SendRequest) (*models.Notification, error) {
	ws, err := s.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("load workspace: %w", err)
	}
	if ws.NotificationsSent >= ws.NotificationsLimit {
		return nil, fmt.Errorf("notification limit reached for plan %q", ws.Plan)
	}

	sub, err := s.store.GetSubscriber(ctx, workspaceID, req.SubscriberID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("subscriber %q not found", req.SubscriberID)
		}
		return nil, fmt.Errorf("load subscriber: %w", err)
	}

	subject := req.Subject
	bodies := map[string]string{}
	chans := req.Channels
	var templateID *string

	if req.TemplateSlug != "" {
		tmpl, err := s.store.GetTemplateBySlug(ctx, workspaceID, req.TemplateSlug)
		if err != nil {
			return nil, fmt.Errorf("load template %q: %w", req.TemplateSlug, err)
		}
		rendered, err := s.tmpl.Render(tmpl, req.Variables)
		if err != nil {
			return nil, err
		}
		subject = rendered.Subject
		bodies = rendered.Bodies
		if len(chans) == 0 {
			chans = tmpl.Channels
		}
		templateID = &tmpl.ID
	} else {
		if len(chans) == 0 {
			return nil, fmt.Errorf("no channels specified")
		}
		if req.Body == "" {
			return nil, fmt.Errorf("body is required when not using a template")
		}
		for _, ch := range chans {
			bodies[ch] = req.Body
		}
	}

	priority := req.Priority
	if priority < 1 || priority > 10 {
		priority = 5
	}

	payload, err := json.Marshal(map[string]interface{}{
		"subject":  subject,
		"bodies":   bodies,
		"metadata": req.Metadata,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	variablesJSON, _ := json.Marshal(req.Variables)

	notif := &models.Notification{
		WorkspaceID:  workspaceID,
		SubscriberID: &sub.ID,
		TemplateID:   templateID,
		Channels:     chans,
		Payload:      payload,
		Variables:    variablesJSON,
		Status:       models.StatusQueued,
		Priority:     priority,
		ScheduledAt:  req.ScheduledAt,
	}
	notif, err = s.store.CreateNotification(ctx, notif)
	if err != nil {
		return nil, err
	}

	for _, ch := range chans {
		if _, ok := s.channels[ch]; !ok {
			s.log.Warn("unknown channel requested, skipping", zap.String("channel", ch))
			continue
		}
		delivery, err := s.store.CreateDelivery(ctx, notif.ID, workspaceID, ch, 3)
		if err != nil {
			return nil, err
		}

		jp, _ := json.Marshal(jobPayload{
			Subject:  subject,
			Body:     bodies[ch],
			Metadata: req.Metadata,
		})
		job := queue.QueueJob{
			DeliveryID:     delivery.ID,
			NotificationID: notif.ID,
			WorkspaceID:    workspaceID,
			Channel:        ch,
			SubscriberID:   sub.ExternalID,
			Payload:        string(jp),
			Attempt:        0,
			MaxAttempts:    delivery.MaxAttempts,
			CreatedAt:      time.Now(),
		}

		if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
			err = s.producer.EnqueueDelayed(ctx, job, time.Until(*req.ScheduledAt))
		} else {
			err = s.producer.Enqueue(ctx, job, priority)
		}
		if err != nil {
			return nil, fmt.Errorf("enqueue %s: %w", ch, err)
		}
		notif.Deliveries = append(notif.Deliveries, *delivery)
	}

	if err := s.store.IncrementNotificationsSent(ctx, workspaceID, 1); err != nil {
		s.log.Warn("increment usage failed", zap.Error(err))
	}
	return notif, nil
}

// SendBulk processes many requests in parallel batches and aggregates errors.
func (s *NotificationService) SendBulk(ctx context.Context, workspaceID string, reqs []SendRequest) error {
	const batchSize = 100
	var (
		mu      sync.Mutex
		errsAll []string
	)
	for start := 0; start < len(reqs); start += batchSize {
		end := start + batchSize
		if end > len(reqs) {
			end = len(reqs)
		}
		var wg sync.WaitGroup
		for _, r := range reqs[start:end] {
			wg.Add(1)
			go func(req SendRequest) {
				defer wg.Done()
				if _, err := s.Send(ctx, workspaceID, req); err != nil {
					mu.Lock()
					errsAll = append(errsAll, fmt.Sprintf("%s: %v", req.SubscriberID, err))
					mu.Unlock()
				}
			}(r)
		}
		wg.Wait()
	}
	if len(errsAll) > 0 {
		return fmt.Errorf("bulk send had %d failures: %v", len(errsAll), errsAll)
	}
	return nil
}

// HandleDelivery is the queue worker handler. Returning an error triggers the
// consumer's retry/backoff logic; returning nil marks the attempt terminal.
func (s *NotificationService) HandleDelivery(ctx context.Context, job queue.QueueJob) error {
	var jp jobPayload
	if err := json.Unmarshal([]byte(job.Payload), &jp); err != nil {
		return fmt.Errorf("unmarshal job payload: %w", err)
	}

	ws, err := s.store.GetWorkspaceByID(ctx, job.WorkspaceID)
	if err != nil {
		return fmt.Errorf("load workspace: %w", err)
	}
	sub, err := s.store.GetSubscriber(ctx, job.WorkspaceID, job.SubscriberID)
	if err != nil {
		// Subscriber removed mid-flight: terminal, do not retry.
		s.markFailed(ctx, job, "subscriber not found")
		return nil
	}
	channel, ok := s.channels[job.Channel]
	if !ok {
		s.markFailed(ctx, job, "unknown channel")
		return nil
	}

	dj := channels.DeliveryJob{
		DeliveryID:  job.DeliveryID,
		WorkspaceID: job.WorkspaceID,
		Subscriber:  sub,
		Workspace:   ws,
		Subject:     jp.Subject,
		Body:        jp.Body,
		Metadata:    jp.Metadata,
	}

	result, sendErr := channel.Send(ctx, dj)

	// Expired web push subscription: clean up and stop retrying.
	if errors.Is(sendErr, channels.ErrSubscriptionGone) {
		if derr := s.store.DeactivateWebPush(ctx, sub.ID); derr != nil {
			s.log.Warn("deactivate web push failed", zap.Error(derr))
		}
		s.markFailed(ctx, job, "subscription gone")
		return nil
	}

	if sendErr != nil {
		// Retryable transport error.
		s.recordAttempt(ctx, job, models.StatusFailed, false, sendErr.Error(), nil)
		s.fireWebhook(ctx, job, EventDeliveryFailed, sendErr.Error())
		return sendErr
	}

	if result != nil && result.Success {
		var pr []byte
		if result.ProviderResponse != nil {
			pr, _ = json.Marshal(result.ProviderResponse)
		}
		s.recordAttempt(ctx, job, models.StatusDelivered, true, "", pr)
		s.fireWebhook(ctx, job, EventDeliveryDelivered, "")
		return nil
	}

	// Best-effort non-success (offline, not configured): terminal, no retry.
	msg := "not delivered"
	if result != nil && result.Error != "" {
		msg = result.Error
	}
	s.recordAttempt(ctx, job, models.StatusFailed, false, msg, nil)
	s.fireWebhook(ctx, job, EventDeliveryFailed, msg)
	return nil
}

func (s *NotificationService) recordAttempt(ctx context.Context, job queue.QueueJob, status string, delivered bool, errMsg string, providerResp []byte) {
	var ep *string
	if errMsg != "" {
		ep = &errMsg
	}
	if err := s.store.MarkDeliveryResult(ctx, job.DeliveryID, status, job.Attempt, ep, providerResp, delivered); err != nil {
		s.log.Error("mark delivery result failed", zap.Error(err))
	}
	s.recomputeNotificationStatus(ctx, job.NotificationID)
}

func (s *NotificationService) markFailed(ctx context.Context, job queue.QueueJob, msg string) {
	s.recordAttempt(ctx, job, models.StatusFailed, false, msg, nil)
}

func (s *NotificationService) recomputeNotificationStatus(ctx context.Context, notificationID string) {
	dels, err := s.store.ListDeliveriesByNotification(ctx, notificationID)
	if err != nil || len(dels) == 0 {
		return
	}
	delivered, failed, pending := 0, 0, 0
	for _, d := range dels {
		switch d.Status {
		case models.StatusDelivered:
			delivered++
		case models.StatusFailed:
			failed++
		default:
			pending++
		}
	}
	status := models.StatusSent
	switch {
	case pending > 0:
		status = models.StatusSent
	case delivered > 0 && failed == 0:
		status = models.StatusDelivered
	case delivered == 0 && failed > 0:
		status = models.StatusFailed
	case delivered > 0 && failed > 0:
		status = models.StatusPartial
	}
	if err := s.store.UpdateNotificationStatus(ctx, notificationID, status); err != nil {
		s.log.Warn("update notification status failed", zap.Error(err))
	}
}

func (s *NotificationService) fireWebhook(ctx context.Context, job queue.QueueJob, event, errMsg string) {
	s.webhooks.Dispatch(ctx, job.WorkspaceID, event, map[string]interface{}{
		"delivery_id":     job.DeliveryID,
		"notification_id": job.NotificationID,
		"channel":         job.Channel,
		"subscriber_id":   job.SubscriberID,
		"error":           errMsg,
	})
}
