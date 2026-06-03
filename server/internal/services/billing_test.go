package services

import (
	"context"
	"testing"

	"github.com/astrazstudio/pushnotify/server/internal/config"
	"go.uber.org/zap"
)

func TestBilling_DisabledWithoutConfig(t *testing.T) {
	s := NewBillingService(nil, config.BillingConfig{}, "http://localhost:3000", zap.NewNop())
	if s.Enabled() {
		t.Fatal("billing should be disabled without a Stripe key")
	}
	if _, err := s.CreateCheckout(context.Background(), "ws_1"); err == nil {
		t.Fatal("expected error when billing is not configured")
	}
	if _, err := s.CreatePortal(context.Background(), "ws_1"); err == nil {
		t.Fatal("expected error when billing is not configured")
	}
}

func TestBilling_EnabledWithConfig(t *testing.T) {
	cfg := config.BillingConfig{StripeSecretKey: "sk_test_x", StripePriceProID: "price_x"}
	s := NewBillingService(nil, cfg, "http://localhost:3000", zap.NewNop())
	if !s.Enabled() {
		t.Fatal("billing should be enabled when key and price are set")
	}
}
