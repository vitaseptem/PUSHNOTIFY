package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
)

// ErrSubscriptionGone signals an expired (410) push subscription; the caller
// should deactivate it.
var ErrSubscriptionGone = fmt.Errorf("web push subscription gone")

// WebPushChannel sends browser push notifications via VAPID.
type WebPushChannel struct{}

// NewWebPushChannel constructs the channel.
func NewWebPushChannel() *WebPushChannel { return &WebPushChannel{} }

func (c *WebPushChannel) Name() string    { return ChannelWebPush }
func (c *WebPushChannel) Validate() error { return nil }

// Send delivers a push message using the workspace VAPID keys and the
// subscriber's stored push subscription.
func (c *WebPushChannel) Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error) {
	if job.Subscriber == nil || len(job.Subscriber.WebPushSubscription) == 0 {
		return &DeliveryResult{Success: false, Error: "no web push subscription"}, nil
	}
	if job.Workspace == nil || job.Workspace.VAPIDPublicKey == "" || job.Workspace.VAPIDPrivateKey == "" {
		return nil, fmt.Errorf("workspace VAPID keys not configured")
	}

	var sub webpush.Subscription
	if err := json.Unmarshal(job.Subscriber.WebPushSubscription, &sub); err != nil {
		return &DeliveryResult{Success: false, Error: "invalid subscription"}, nil
	}

	url, _ := job.Metadata["url"].(string)
	icon, _ := job.Metadata["icon"].(string)
	notif := map[string]interface{}{
		"title": job.Subject,
		"body":  job.Body,
		"icon":  icon,
		"url":   url,
	}
	payload, err := json.Marshal(notif)
	if err != nil {
		return nil, fmt.Errorf("marshal webpush payload: %w", err)
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payload, &sub, &webpush.Options{
		Subscriber:      job.Workspace.Slug + "@pushnotify.dev",
		VAPIDPublicKey:  job.Workspace.VAPIDPublicKey,
		VAPIDPrivateKey: job.Workspace.VAPIDPrivateKey,
		TTL:             60,
	})
	if err != nil {
		return nil, fmt.Errorf("send webpush: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound:
		// 410/404: subscription is dead. Don't retry; signal for cleanup.
		return &DeliveryResult{Success: false, Error: "subscription gone"}, ErrSubscriptionGone
	case resp.StatusCode == http.StatusTooManyRequests:
		// 429: rate limited upstream — return an error to trigger a retry.
		return nil, fmt.Errorf("webpush rate limited: %s", string(body))
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return &DeliveryResult{Success: true, ProviderID: "webpush", ProviderResponse: string(body)}, nil
	default:
		return nil, fmt.Errorf("webpush failed status=%d body=%s", resp.StatusCode, string(body))
	}
}
