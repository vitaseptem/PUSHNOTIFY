package handlers

import (
	"io"
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"go.uber.org/zap"
)

// BillingStatus reports whether billing is enabled (so the dashboard can show
// or hide the upgrade button).
func (h *Handlers) BillingStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": h.Billing != nil && h.Billing.Enabled()})
}

// BillingCheckout creates a Stripe Checkout session and returns its URL.
func (h *Handlers) BillingCheckout(w http.ResponseWriter, r *http.Request) {
	if h.Billing == nil || !h.Billing.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "billing is not configured")
		return
	}
	url, err := h.Billing.CreateCheckout(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// BillingPortal creates a Stripe Customer Portal session and returns its URL.
func (h *Handlers) BillingPortal(w http.ResponseWriter, r *http.Request) {
	if h.Billing == nil || !h.Billing.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "billing is not configured")
		return
	}
	url, err := h.Billing.CreatePortal(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

// BillingWebhook receives Stripe webhook events. It is public (no JWT/API key)
// and authenticated instead via the Stripe signature on the raw body.
func (h *Handlers) BillingWebhook(w http.ResponseWriter, r *http.Request) {
	if h.Billing == nil {
		writeError(w, http.StatusServiceUnavailable, "billing is not configured")
		return
	}
	const maxBody = 1 << 20 // 1 MiB
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	if err := h.Billing.HandleWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature")); err != nil {
		h.Log.Warn("stripe webhook rejected", zap.Error(err))
		writeError(w, http.StatusBadRequest, "webhook error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"received": true})
}
