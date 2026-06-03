package handlers

import (
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/hub"
	"github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/astrazstudio/pushnotify/server/internal/token"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Origin checking is delegated to the JWT: a valid subscriber token is
	// required regardless of Origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket upgrades the connection and registers the subscriber in the hub.
// Auth is via a subscriber JWT passed as the `token` query parameter.
func (h *Handlers) WebSocket(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspaceID")
	subscriberID := chi.URLParam(r, "subscriberID")

	tok := r.URL.Query().Get("token")
	if tok == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	claims, err := token.ParseSubscriber(h.Cfg.JWT.Secret, tok)
	if err != nil || claims.WorkspaceID != workspaceID || claims.SubscriberID != subscriberID {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Log.Debug("ws upgrade failed", zap.Error(err))
		return
	}

	client := hub.NewClient(h.Hub, conn, workspaceID, subscriberID, r.UserAgent())
	client.Start()
}

// IssueSubscriberToken mints a short-lived WebSocket token for a subscriber.
// It is authenticated by the workspace API key / JWT.
func (h *Handlers) IssueSubscriberToken(w http.ResponseWriter, r *http.Request) {
	type req struct {
		SubscriberID string `json:"subscriber_id"`
	}
	var body req
	if err := decode(r, &body); err != nil || body.SubscriberID == "" {
		writeError(w, http.StatusBadRequest, "subscriber_id is required")
		return
	}
	workspaceID := middleware.WorkspaceID(r.Context())
	tok, err := token.IssueSubscriber(h.Cfg.JWT.Secret, workspaceID, body.SubscriberID, h.Cfg.JWT.Expiry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token issue failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}
