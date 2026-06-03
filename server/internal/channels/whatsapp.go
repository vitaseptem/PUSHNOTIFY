package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/config"
)

// WhatsAppChannel sends text messages through an Evolution API (or Z-API
// compatible) HTTP gateway.
type WhatsAppChannel struct {
	cfg    config.WhatsAppConfig
	client *http.Client
}

// NewWhatsAppChannel constructs the channel.
func NewWhatsAppChannel(cfg config.WhatsAppConfig) *WhatsAppChannel {
	return &WhatsAppChannel{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *WhatsAppChannel) Name() string { return ChannelWhatsApp }

func (c *WhatsAppChannel) Validate() error {
	if c.cfg.APIURL == "" {
		return fmt.Errorf("whatsapp channel: Evolution API URL not configured")
	}
	return nil
}

// Send posts a text message to the gateway's /message/sendText endpoint.
func (c *WhatsAppChannel) Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error) {
	if c.cfg.APIURL == "" {
		return &DeliveryResult{Success: false, Error: "whatsapp not configured"}, nil
	}
	if job.Subscriber == nil || job.Subscriber.WhatsApp == nil || *job.Subscriber.WhatsApp == "" {
		return &DeliveryResult{Success: false, Error: "subscriber has no whatsapp"}, nil
	}

	reqBody := map[string]interface{}{
		"number": *job.Subscriber.WhatsApp,
		"text":   job.Body,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal whatsapp body: %w", err)
	}

	url := strings.TrimRight(c.cfg.APIURL, "/") + "/message/sendText"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("build whatsapp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", c.cfg.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("whatsapp request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &DeliveryResult{Success: true, ProviderID: "evolution", ProviderResponse: string(body)}, nil
	}
	// 5xx and session-disconnected errors are retryable.
	return nil, fmt.Errorf("whatsapp send failed status=%d body=%s", resp.StatusCode, string(body))
}
