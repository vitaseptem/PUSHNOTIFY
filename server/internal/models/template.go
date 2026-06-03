package models

import "time"

// Template defines reusable per-channel message bodies with variables.
type Template struct {
	ID            string    `json:"id"`
	WorkspaceID   string    `json:"workspace_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Channels      []string  `json:"channels"`
	Subject       string    `json:"subject"`
	BodyWebSocket string    `json:"body_websocket"`
	BodyWebPush   string    `json:"body_webpush"`
	BodyEmail     string    `json:"body_email"`
	BodyWhatsApp  string    `json:"body_whatsapp"`
	BodySMS       string    `json:"body_sms"`
	Variables     []string  `json:"variables"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// BodyForChannel returns the template body configured for a given channel.
func (t *Template) BodyForChannel(channel string) string {
	switch channel {
	case "websocket":
		return t.BodyWebSocket
	case "webpush":
		return t.BodyWebPush
	case "email":
		return t.BodyEmail
	case "whatsapp":
		return t.BodyWhatsApp
	case "sms":
		return t.BodySMS
	default:
		return ""
	}
}
