package handlers

import (
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"github.com/astrazstudio/pushnotify/server/pkg/paginator"
	"github.com/go-chi/chi/v5"
)

// SendNotification enqueues a notification across the requested channels.
func (h *Handlers) SendNotification(w http.ResponseWriter, r *http.Request) {
	var req services.SendRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	notif, err := h.Notifier.Send(r.Context(), middleware.WorkspaceID(r.Context()), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, notif)
}

type bulkRequest struct {
	Notifications []services.SendRequest `json:"notifications"`
}

// SendBulkNotifications enqueues many notifications in batches.
func (h *Handlers) SendBulkNotifications(w http.ResponseWriter, r *http.Request) {
	var req bulkRequest
	if err := decode(r, &req); err != nil || len(req.Notifications) == 0 {
		writeError(w, http.StatusBadRequest, "notifications array is required")
		return
	}
	if err := h.Notifier.SendBulk(r.Context(), middleware.WorkspaceID(r.Context()), req.Notifications); err != nil {
		writeJSON(w, http.StatusMultiStatus, map[string]interface{}{
			"accepted": len(req.Notifications),
			"error":    err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"accepted": len(req.Notifications)})
}

// ListNotifications returns a paginated, optionally status-filtered list.
func (h *Handlers) ListNotifications(w http.ResponseWriter, r *http.Request) {
	p := paginator.FromRequest(r)
	status := r.URL.Query().Get("status")
	items, total, err := h.Store.ListNotifications(r.Context(), middleware.WorkspaceID(r.Context()), status, p.Limit(), p.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, paginator.NewResult(items, p, total))
}

// GetNotification returns a single notification with its deliveries.
func (h *Handlers) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ws := middleware.WorkspaceID(r.Context())
	notif, err := h.Store.GetNotificationByID(r.Context(), ws, id)
	if err != nil {
		writeError(w, statusForStoreError(err), "notification not found")
		return
	}
	deliveries, err := h.Store.ListDeliveriesByNotification(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load deliveries failed")
		return
	}
	notif.Deliveries = deliveries
	writeJSON(w, http.StatusOK, notif)
}

// GetNotificationDeliveries returns just the deliveries for a notification.
func (h *Handlers) GetNotificationDeliveries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ws := middleware.WorkspaceID(r.Context())
	if _, err := h.Store.GetNotificationByID(r.Context(), ws, id); err != nil {
		writeError(w, statusForStoreError(err), "notification not found")
		return
	}
	deliveries, err := h.Store.ListDeliveriesByNotification(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "load deliveries failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": deliveries})
}
