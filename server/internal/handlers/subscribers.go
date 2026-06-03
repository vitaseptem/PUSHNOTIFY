package handlers

import (
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"github.com/astrazstudio/pushnotify/server/pkg/paginator"
	"github.com/go-chi/chi/v5"
)

// ListSubscribers returns a paginated list of subscribers.
func (h *Handlers) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	p := paginator.FromRequest(r)
	items, total, err := h.Store.ListSubscribers(r.Context(), middleware.WorkspaceID(r.Context()), p.Limit(), p.Offset())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, paginator.NewResult(items, p, total))
}

// UpsertSubscriber creates or updates a subscriber.
func (h *Handlers) UpsertSubscriber(w http.ResponseWriter, r *http.Request) {
	var in services.SubscriberInput
	if err := decode(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sub, err := h.Subscribers.Upsert(r.Context(), middleware.WorkspaceID(r.Context()), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// GetSubscriber returns a subscriber by external id.
func (h *Handlers) GetSubscriber(w http.ResponseWriter, r *http.Request) {
	sub, err := h.Store.GetSubscriber(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "externalId"))
	if err != nil {
		writeError(w, statusForStoreError(err), "subscriber not found")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// UpdateSubscriber updates a subscriber identified by the path param.
func (h *Handlers) UpdateSubscriber(w http.ResponseWriter, r *http.Request) {
	var in services.SubscriberInput
	if err := decode(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	in.ExternalID = chi.URLParam(r, "externalId")
	sub, err := h.Subscribers.Upsert(r.Context(), middleware.WorkspaceID(r.Context()), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// DeleteSubscriber removes a subscriber.
func (h *Handlers) DeleteSubscriber(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.DeleteSubscriber(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "externalId")); err != nil {
		writeError(w, statusForStoreError(err), "delete failed")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// RegisterWebPush stores a browser push subscription for a subscriber.
func (h *Handlers) RegisterWebPush(w http.ResponseWriter, r *http.Request) {
	var sub services.WebPushSubscription
	if err := decode(r, &sub); err != nil {
		writeError(w, http.StatusBadRequest, "invalid subscription")
		return
	}
	err := h.Subscribers.RegisterWebPush(r.Context(), middleware.WorkspaceID(r.Context()), chi.URLParam(r, "externalId"), sub)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
}
