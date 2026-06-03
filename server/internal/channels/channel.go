package channels

import (
	"context"

	"github.com/astrazstudio/pushnotify/server/internal/models"
)

// Channel names.
const (
	ChannelWebSocket = "websocket"
	ChannelWebPush   = "webpush"
	ChannelEmail     = "email"
	ChannelWhatsApp  = "whatsapp"
	ChannelSMS       = "sms"
)

// DeliveryJob carries everything a channel needs to deliver one message.
type DeliveryJob struct {
	DeliveryID  string
	WorkspaceID string
	Subscriber  *models.Subscriber
	Workspace   *models.Workspace
	Subject     string
	Body        string
	Metadata    map[string]interface{}
}

// DeliveryResult is the outcome of a single send attempt.
type DeliveryResult struct {
	Success          bool        `json:"success"`
	ProviderID       string      `json:"provider_id,omitempty"`
	ProviderResponse interface{} `json:"provider_response,omitempty"`
	Error            string      `json:"error,omitempty"`
}

// Channel is the plugable delivery transport interface.
type Channel interface {
	Name() string
	Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error)
	Validate() error
}
