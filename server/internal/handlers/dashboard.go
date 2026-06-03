package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/middleware"
)

// DashboardOverview returns the cached analytics overview.
func (h *Handlers) DashboardOverview(w http.ResponseWriter, r *http.Request) {
	ov, err := h.Analytics.GetOverview(r.Context(), middleware.WorkspaceID(r.Context()))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "overview failed")
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

// DashboardAnalytics returns a period-scoped analytics payload.
func (h *Handlers) DashboardAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	data, err := h.Analytics.GetAnalytics(r.Context(), middleware.WorkspaceID(r.Context()), period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "analytics failed")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// DashboardLive streams live metrics via Server-Sent Events. It emits the
// current connected count and the cached overview every few seconds.
func (h *Handlers) DashboardLive(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	workspaceID := middleware.WorkspaceID(r.Context())
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// Send an initial event immediately.
	h.writeSSE(w, flusher, workspaceID)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			h.writeSSE(w, flusher, workspaceID)
		}
	}
}

func (h *Handlers) writeSSE(w http.ResponseWriter, flusher http.Flusher, workspaceID string) {
	payload := map[string]interface{}{
		"connected_now": h.Hub.GetConnectedCount(workspaceID),
		"timestamp":     time.Now().UTC(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// Health is a simple liveness probe.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
