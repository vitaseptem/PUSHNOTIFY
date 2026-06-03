package services

import "testing"

import "github.com/astrazstudio/pushnotify/server/internal/models"

func TestRenderSubstitutesVariables(t *testing.T) {
	svc := NewTemplateService()
	tmpl := &models.Template{
		Channels:  []string{"email", "sms"},
		Subject:   "Order {{order_id}}",
		BodyEmail: "Hi {{name}}, your order {{order_id}} is confirmed.",
		BodySMS:   "Order {{order_id}} confirmed",
		Variables: []string{"name", "order_id"},
	}
	rc, err := svc.Render(tmpl, map[string]string{"name": "Ana", "order_id": "1234"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if rc.Subject != "Order 1234" {
		t.Errorf("subject = %q", rc.Subject)
	}
	if rc.Bodies["email"] != "Hi Ana, your order 1234 is confirmed." {
		t.Errorf("email body = %q", rc.Bodies["email"])
	}
	if rc.Bodies["sms"] != "Order 1234 confirmed" {
		t.Errorf("sms body = %q", rc.Bodies["sms"])
	}
}

func TestRenderMissingVariableFails(t *testing.T) {
	svc := NewTemplateService()
	tmpl := &models.Template{
		Channels:  []string{"email"},
		BodyEmail: "Hi {{name}}",
		Variables: []string{"name"},
	}
	if _, err := svc.Render(tmpl, map[string]string{}); err == nil {
		t.Fatal("expected error for missing variable")
	}
}

func TestExtractVariables(t *testing.T) {
	got := ExtractVariables("{{a}} and {{ b }}", "{{a}} again")
	if len(got) != 2 {
		t.Fatalf("expected 2 unique vars, got %v", got)
	}
}
