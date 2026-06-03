package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/config"
	"github.com/astrazstudio/pushnotify/server/internal/models"
	stripe "github.com/stripe/stripe-go/v79"
	portalsession "github.com/stripe/stripe-go/v79/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/webhook"
	"go.uber.org/zap"
)

// Plans and their monthly notification limits.
const (
	PlanFree  = "free"
	PlanPro   = "pro"
	FreeLimit = 10000
	ProLimit  = 500000
)

// BillingStore is the subset of the store the billing service needs.
type BillingStore interface {
	GetWorkspaceByID(ctx context.Context, id string) (*models.Workspace, error)
	SetWorkspaceStripeCustomer(ctx context.Context, workspaceID, customerID string) error
	GetWorkspaceByStripeCustomer(ctx context.Context, customerID string) (*models.Workspace, error)
	UpdateWorkspaceSubscription(ctx context.Context, workspaceID, plan string, limit int64, status string, subscriptionID *string, periodEnd *time.Time) error
}

// BillingService integrates Stripe Checkout, the Customer Portal and webhooks.
type BillingService struct {
	store  BillingStore
	cfg    config.BillingConfig
	appURL string
	log    *zap.Logger
}

// NewBillingService constructs the service and sets the global Stripe key.
func NewBillingService(store BillingStore, cfg config.BillingConfig, appURL string, log *zap.Logger) *BillingService {
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}
	return &BillingService{store: store, cfg: cfg, appURL: appURL, log: log}
}

// Enabled reports whether Stripe billing is configured.
func (s *BillingService) Enabled() bool { return s.cfg.Enabled() }

// CreateCheckout returns a Stripe Checkout URL for upgrading to Pro.
func (s *BillingService) CreateCheckout(ctx context.Context, workspaceID string) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("billing is not configured")
	}
	ws, err := s.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return "", fmt.Errorf("load workspace: %w", err)
	}
	customerID, err := s.ensureCustomer(ctx, ws)
	if err != nil {
		return "", err
	}

	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(customerID),
		SuccessURL: stripe.String(s.appURL + "/settings?billing=success"),
		CancelURL:  stripe.String(s.appURL + "/settings?billing=cancel"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(s.cfg.StripePriceProID), Quantity: stripe.Int64(1)},
		},
	}
	params.Metadata = map[string]string{"workspace_id": ws.ID}

	sess, err := checkoutsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}

// CreatePortal returns a Stripe Customer Portal URL for managing the subscription.
func (s *BillingService) CreatePortal(ctx context.Context, workspaceID string) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("billing is not configured")
	}
	ws, err := s.store.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return "", fmt.Errorf("load workspace: %w", err)
	}
	if ws.StripeCustomerID == nil || *ws.StripeCustomerID == "" {
		return "", fmt.Errorf("no billing customer for workspace")
	}
	params := &stripe.BillingPortalSessionParams{
		Customer:  ws.StripeCustomerID,
		ReturnURL: stripe.String(s.appURL + "/settings"),
	}
	ps, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}
	return ps.URL, nil
}

// HandleWebhook verifies and processes a Stripe webhook event, keeping the
// workspace plan in sync with the subscription state.
func (s *BillingService) HandleWebhook(ctx context.Context, payload []byte, signature string) error {
	event, err := webhook.ConstructEvent(payload, signature, s.cfg.StripeWebhookSecret)
	if err != nil {
		return fmt.Errorf("verify webhook signature: %w", err)
	}

	switch string(event.Type) {
	case "checkout.session.completed":
		var cs stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &cs); err != nil {
			return fmt.Errorf("decode checkout session: %w", err)
		}
		if cs.Customer == nil {
			return nil
		}
		var subID string
		if cs.Subscription != nil {
			subID = cs.Subscription.ID
		}
		return s.activatePro(ctx, cs.Customer.ID, subID, nil)

	case "customer.subscription.created", "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("decode subscription: %w", err)
		}
		if sub.Customer == nil {
			return nil
		}
		periodEnd := time.Unix(sub.CurrentPeriodEnd, 0)
		if sub.Status == stripe.SubscriptionStatusActive || sub.Status == stripe.SubscriptionStatusTrialing {
			return s.activatePro(ctx, sub.Customer.ID, sub.ID, &periodEnd)
		}
		return s.downgrade(ctx, sub.Customer.ID, string(sub.Status))

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return fmt.Errorf("decode subscription: %w", err)
		}
		if sub.Customer == nil {
			return nil
		}
		return s.downgrade(ctx, sub.Customer.ID, "canceled")

	default:
		// Unhandled event types are acknowledged with no action.
		return nil
	}
}

func (s *BillingService) ensureCustomer(ctx context.Context, ws *models.Workspace) (string, error) {
	if ws.StripeCustomerID != nil && *ws.StripeCustomerID != "" {
		return *ws.StripeCustomerID, nil
	}
	params := &stripe.CustomerParams{Name: stripe.String(ws.Name)}
	params.Metadata = map[string]string{"workspace_id": ws.ID}
	c, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe customer: %w", err)
	}
	if err := s.store.SetWorkspaceStripeCustomer(ctx, ws.ID, c.ID); err != nil {
		return "", fmt.Errorf("persist stripe customer: %w", err)
	}
	return c.ID, nil
}

func (s *BillingService) activatePro(ctx context.Context, customerID, subscriptionID string, periodEnd *time.Time) error {
	ws, err := s.store.GetWorkspaceByStripeCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("workspace for customer %s: %w", customerID, err)
	}
	var subPtr *string
	if subscriptionID != "" {
		subPtr = &subscriptionID
	}
	if err := s.store.UpdateWorkspaceSubscription(ctx, ws.ID, PlanPro, ProLimit, "active", subPtr, periodEnd); err != nil {
		return err
	}
	s.log.Info("workspace upgraded to pro", zap.String("workspace", ws.ID))
	return nil
}

func (s *BillingService) downgrade(ctx context.Context, customerID, status string) error {
	ws, err := s.store.GetWorkspaceByStripeCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("workspace for customer %s: %w", customerID, err)
	}
	if err := s.store.UpdateWorkspaceSubscription(ctx, ws.ID, PlanFree, FreeLimit, status, nil, nil); err != nil {
		return err
	}
	s.log.Info("workspace downgraded to free", zap.String("workspace", ws.ID), zap.String("status", status))
	return nil
}
