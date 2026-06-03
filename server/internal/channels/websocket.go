package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/hub"
)

// WebSocketChannel delivers notifications to live browser connections through
// the hub's Redis bridge, so it works whether the worker runs in-process with
// the API or in a separate process. It is best-effort: an offline subscriber is
// not treated as a retryable error.
type WebSocketChannel struct {
	hub *hub.Hub
}

// NewWebSocketChannel wires the channel to the hub.
func NewWebSocketChannel(h *hub.Hub) *WebSocketChannel {
	return &WebSocketChannel{hub: h}
}

func (c *WebSocketChannel) Name() string { return ChannelWebSocket }

func (c *WebSocketChannel) Validate() error {
	if c.hub == nil {
		return fmt.Errorf("websocket channel: hub not configured")
	}
	return nil
}

// Send publishes the payload to the subscriber's live connections. If the
// subscriber is offline it reports success=false with an "offline" error but
// returns a nil error so the queue does not retry (WebSocket is best-effort).
func (c *WebSocketChannel) Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error) {
	payload := map[string]interface{}{
		"id":         job.DeliveryID,
		"type":       "notification",
		"subject":    job.Subject,
		"body":       job.Body,
		"metadata":   job.Metadata,
		"created_at": time.Now().UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal ws payload: %w", err)
	}

	online, err := c.hub.PublishToSubscriber(ctx, job.WorkspaceID, job.Subscriber.ExternalID, data)
	if err != nil {
		return nil, fmt.Errorf("publish ws: %w", err)
	}
	if !online {
		return &DeliveryResult{Success: false, Error: "offline"}, nil
	}
	return &DeliveryResult{Success: true, ProviderID: "ws"}, nil
}
