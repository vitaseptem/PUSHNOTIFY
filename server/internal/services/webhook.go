package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Webhook event names.
const (
	EventDeliverySent      = "delivery.sent"
	EventDeliveryDelivered = "delivery.delivered"
	EventDeliveryFailed    = "delivery.failed"
)

// WebhookService dispatches signed status webhooks to customer endpoints.
type WebhookService struct {
	store  Datastore
	log    *zap.Logger
	client *http.Client
}

// NewWebhookService constructs the service.
func NewWebhookService(st Datastore, log *zap.Logger) *WebhookService {
	return &WebhookService{
		store:  st,
		log:    log,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Dispatch sends the event to all active webhooks subscribed to it. Failures
// are logged but never block the caller (fire-and-forget per webhook).
func (s *WebhookService) Dispatch(ctx context.Context, workspaceID, event string, payload map[string]interface{}) {
	hooks, err := s.store.ListActiveWebhooksForEvent(ctx, workspaceID, event)
	if err != nil {
		s.log.Warn("list webhooks for dispatch failed", zap.Error(err))
		return
	}
	if len(hooks) == 0 {
		return
	}

	body := map[string]interface{}{
		"event":     event,
		"data":      payload,
		"timestamp": time.Now().UTC(),
	}
	raw, err := json.Marshal(body)
	if err != nil {
		s.log.Error("marshal webhook payload failed", zap.Error(err))
		return
	}

	for _, h := range hooks {
		s.send(ctx, h.ID, h.URL, h.Secret, event, raw)
	}
}

func (s *WebhookService) send(ctx context.Context, webhookID, url, secret, event string, raw []byte) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		s.log.Warn("build webhook request failed", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PushNotify-Event", event)
	req.Header.Set("X-PushNotify-Signature", sign(secret, raw))

	resp, err := s.client.Do(req)
	if err != nil {
		s.log.Warn("webhook delivery failed", zap.String("url", url), zap.Error(err))
		_ = s.store.CreateWebhookLog(ctx, webhookID, event, raw, nil, strPtr(err.Error()))
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	status := resp.StatusCode
	bodyStr := string(respBody)
	if err := s.store.CreateWebhookLog(ctx, webhookID, event, raw, &status, &bodyStr); err != nil {
		s.log.Warn("record webhook log failed", zap.Error(err))
	}
}

// sign returns the hex HMAC-SHA256 of the payload using the webhook secret.
func sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func strPtr(s string) *string { return &s }
