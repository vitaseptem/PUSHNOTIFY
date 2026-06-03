package handlers

import (
	"net/http"
	"strconv"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
	"github.com/go-chi/chi/v5"
)

type createWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
}

// ListWebhooks returns the workspace's webhooks.
func (h *Handlers) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := h.Store.ListWebhooks(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": hooks})
}

// CreateWebhook registers a new webhook endpoint with a generated secret.
func (h *Handlers) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := decode(r, &req); err != nil || req.URL == "" || len(req.Events) == 0 {
		writeError(w, http.StatusBadRequest, "url and events are required")
		return
	}
	secret, err := crypto.RandomSecret(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "secret generation failed")
		return
	}
	hook, err := h.Store.CreateWebhook(r.Context(), middleware.WorkspaceID(r.Context()), req.URL, secret, req.Events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	// Reveal the secret once so the customer can verify signatures.
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"webhook": hook,
		"secret":  secret,
	})
}

// DeleteWebhook removes a webhook.
func (h *Handlers) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.DeleteWebhook(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "id")); err != nil {
		writeError(w, statusForStoreError(err), "delete failed")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// WebhookLogs returns recent dispatch logs for a webhook.
func (h *Handlers) WebhookLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	logs, err := h.Store.ListWebhookLogs(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "id"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": logs})
}
