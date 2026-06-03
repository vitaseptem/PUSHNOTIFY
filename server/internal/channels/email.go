package channels

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/astrazstudio/pushnotify/server/internal/config"
)

// EmailChannel sends notifications over SMTP using a premium dark HTML shell.
type EmailChannel struct {
	cfg config.EmailConfig
}

// NewEmailChannel constructs the channel with global SMTP defaults.
func NewEmailChannel(cfg config.EmailConfig) *EmailChannel {
	return &EmailChannel{cfg: cfg}
}

func (c *EmailChannel) Name() string { return ChannelEmail }

func (c *EmailChannel) Validate() error {
	if c.cfg.Host == "" {
		return fmt.Errorf("email channel: SMTP host not configured")
	}
	return nil
}

// Send composes and dispatches an HTML email to the subscriber.
func (c *EmailChannel) Send(ctx context.Context, job DeliveryJob) (*DeliveryResult, error) {
	if c.cfg.Host == "" {
		return &DeliveryResult{Success: false, Error: "smtp not configured"}, nil
	}
	if job.Subscriber == nil || job.Subscriber.Email == nil || *job.Subscriber.Email == "" {
		return &DeliveryResult{Success: false, Error: "subscriber has no email"}, nil
	}

	to := *job.Subscriber.Email
	subject := job.Subject
	if subject == "" {
		subject = "Notification"
	}
	htmlBody := renderEmailHTML(subject, job.Body)

	msg := buildMIME(c.cfg.From, to, subject, htmlBody)

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var auth smtp.Auth
	if c.cfg.User != "" {
		auth = smtp.PlainAuth("", c.cfg.User, c.cfg.Pass, c.cfg.Host)
	}

	// smtp.SendMail does not accept a context; we honor cancellation by
	// checking before dialing. The SMTP library enforces its own timeouts.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := smtp.SendMail(addr, auth, extractAddr(c.cfg.From), []string{to}, []byte(msg)); err != nil {
		return nil, fmt.Errorf("smtp send: %w", err)
	}
	return &DeliveryResult{Success: true, ProviderID: "smtp"}, nil
}

func buildMIME(from, to, subject, htmlBody string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return b.String()
}

func extractAddr(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.LastIndex(from, ">"); j > i {
			return from[i+1 : j]
		}
	}
	return from
}

func renderEmailHTML(subject, body string) string {
	return `<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#0A0A0F;font-family:Inter,Arial,sans-serif;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#0A0A0F;padding:32px 0;">
    <tr><td align="center">
      <table role="presentation" width="560" cellpadding="0" cellspacing="0"
        style="background:#111118;border:1px solid #1E1E2E;border-radius:16px;overflow:hidden;">
        <tr><td style="padding:28px 32px;border-bottom:1px solid #1E1E2E;">
          <span style="color:#F1F1F5;font-size:18px;font-weight:700;">push<span style="color:#7B1C2E;">notify</span></span>
        </td></tr>
        <tr><td style="padding:32px;">
          <h1 style="margin:0 0 16px;color:#F1F1F5;font-size:20px;">` + htmlEscape(subject) + `</h1>
          <div style="color:#8B8BA0;font-size:15px;line-height:1.6;">` + body + `</div>
        </td></tr>
        <tr><td style="padding:20px 32px;border-top:1px solid #1E1E2E;color:#8B8BA0;font-size:12px;">
          Sent via PushNotify · Built by Astraz Studio
        </td></tr>
      </table>
    </td></tr>
  </table>
</body></html>`
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
