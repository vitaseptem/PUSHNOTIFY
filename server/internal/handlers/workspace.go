package handlers

import (
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
	"github.com/go-chi/chi/v5"
)

// GetWorkspace returns the authenticated workspace.
func (h *Handlers) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, err := h.Store.GetWorkspaceByID(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, statusForStoreError(err), "workspace not found")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

type updateWorkspaceRequest struct {
	Name string `json:"name"`
}

// UpdateWorkspace updates the workspace name.
func (h *Handlers) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req updateWorkspaceRequest
	if err := decode(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	ws, err := h.Store.UpdateWorkspace(r.Context(), middleware.WorkspaceID(r.Context()), req.Name)
	if err != nil {
		writeError(w, statusForStoreError(err), "update failed")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

// CreateAPIKey generates a new API key (plaintext shown once).
func (h *Handlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := decode(r, &req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	plain, hash, prefix, err := crypto.GenerateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "generate failed")
		return
	}
	key, err := h.Store.CreateAPIKey(r.Context(), middleware.WorkspaceID(r.Context()), req.Name, hash, prefix)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	key.PlainKey = plain // returned once, never stored
	writeJSON(w, http.StatusCreated, key)
}

// ListAPIKeys returns the workspace's API keys (without secrets).
func (h *Handlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.Store.ListAPIKeys(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": keys})
}

// DeleteAPIKey revokes an API key.
func (h *Handlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.Store.DeleteAPIKey(r.Context(), middleware.WorkspaceID(r.Context()), id); err != nil {
		writeError(w, statusForStoreError(err), "delete failed")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Usage returns plan usage versus the limit.
func (h *Handlers) Usage(w http.ResponseWriter, r *http.Request) {
	ws, err := h.Store.GetWorkspaceByID(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, statusForStoreError(err), "workspace not found")
		return
	}
	pct := 0.0
	if ws.NotificationsLimit > 0 {
		pct = float64(ws.NotificationsSent) / float64(ws.NotificationsLimit) * 100
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plan":                ws.Plan,
		"notifications_sent":  ws.NotificationsSent,
		"notifications_limit": ws.NotificationsLimit,
		"usage_percent":       pct,
	})
}

// RegenerateVAPID rotates the workspace VAPID keypair.
func (h *Handlers) RegenerateVAPID(w http.ResponseWriter, r *http.Request) {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vapid generation failed")
		return
	}
	if err := h.Store.UpdateWorkspaceVAPID(r.Context(), middleware.WorkspaceID(r.Context()), pub, priv); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"vapid_public_key": pub})
}
