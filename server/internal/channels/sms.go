package channels

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/config"
)

// SMSChannel sends SMS via the Twilio REST API. It is plugable: swapping the
// endpoint/credentials is enough to target a different provider.
type SMSChannel struct {
	cfg    config.SMSConfig
	client *http.Client
}

// NewSMSChannel constructs the channel.
func NewSMSChannel(cfg config.SMSConfig) *SMSChannel {
	return &SMSChannel{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *SMSChannel) Name() string { return ChannelSMS }

func (c *SMSChannel) Validate() error {
	if c.cfg.TwilioSID == "" {
		return fmt.Errorf("sms channel: Twilio SID not configured")
	}
	return nil
}

// Send dispatches an SMS through Twilio's Messages resource.
func (c *SMSChannel) Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error) {
	if c.cfg.TwilioSID == "" || c.cfg.TwilioToken == "" {
		return &DeliveryResult{Success: false, Error: "sms not configured"}, nil
	}
	if job.Subscriber == nil || job.Subscriber.Phone == nil || *job.Subscriber.Phone == "" {
		return &DeliveryResult{Success: false, Error: "subscriber has no phone"}, nil
	}

	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.cfg.TwilioSID)
	form := url.Values{}
	form.Set("To", *job.Subscriber.Phone)
	form.Set("From", c.cfg.TwilioFrom)
	form.Set("Body", job.Body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build sms request: %w", err)
	}
	req.SetBasicAuth(c.cfg.TwilioSID, c.cfg.TwilioToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sms request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &DeliveryResult{Success: true, ProviderID: "twilio", ProviderResponse: string(body)}, nil
	}
	return nil, fmt.Errorf("sms send failed status=%d body=%s", resp.StatusCode, string(body))
}
