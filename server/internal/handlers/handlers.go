// Package handlers implements the HTTP API endpoints.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/astrazstudio/pushnotify/server/internal/config"
	"github.com/astrazstudio/pushnotify/server/internal/hub"
	"github.com/astrazstudio/pushnotify/server/internal/services"
	"github.com/astrazstudio/pushnotify/server/internal/store"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Handlers bundles every dependency the HTTP layer needs.
type Handlers struct {
	Cfg         *config.Config
	Store       DataStore
	Redis       *redis.Client
	Hub         *hub.Hub
	Notifier    *services.NotificationService
	Subscribers *services.SubscriberService
	Analytics   *services.AnalyticsService
	Templates   *services.TemplateService
	Log         *zap.Logger
}

// writeJSON encodes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			// Response already committed; nothing actionable beyond logging upstream.
			return
		}
	}
}

// writeError writes a JSON error body.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// decode reads and validates a JSON request body into v.
func decode(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// statusForStoreError maps store errors to HTTP status codes.
func statusForStoreError(err error) int {
	if errors.Is(err, store.ErrNotFound) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
