// Package router wires every HTTP route together.
package router

import (
	"net/http"
	"time"

	"github.com/astrazstudio/pushnotify/server/internal/handlers"
	mw "github.com/astrazstudio/pushnotify/server/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// New builds the full chi router.
func New(h *handlers.Handlers, rdb *redis.Client, jwtSecret string, log *zap.Logger) http.Handler {
	r := chi.NewRouter()
	rl := mw.NewRateLimiter(rdb)

	// Global middleware.
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(mw.Logger(log))
	r.Use(mw.CORS("*"))

	r.Get("/health", h.Health)

	// WebSocket endpoint: authenticated via the `token` query param.
	r.Get("/ws/{workspaceID}/{subscriberID}", h.WebSocket)

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes, rate limited per IP.
		r.Group(func(r chi.Router) {
			r.Use(rl.ByIP(100, time.Minute))
			r.Post("/auth/register", h.Register)
			r.Post("/auth/login", h.Login)
		})

		// Authenticated routes (JWT). Workspace-scoped rate limit.
		r.Group(func(r chi.Router) {
			r.Use(mw.JWTAuth(jwtSecret))
			r.Use(rl.ByWorkspace(1000, time.Minute))

			r.Post("/auth/refresh", h.Refresh)
			r.Get("/auth/me", h.Me)

			// Workspace.
			r.Get("/workspace", h.GetWorkspace)
			r.Put("/workspace", h.UpdateWorkspace)
			r.Post("/workspace/api-keys", h.CreateAPIKey)
			r.Get("/workspace/api-keys", h.ListAPIKeys)
			r.Delete("/workspace/api-keys/{id}", h.DeleteAPIKey)
			r.Get("/workspace/usage", h.Usage)
			r.Post("/workspace/vapid/regenerate", h.RegenerateVAPID)

			// Templates.
			r.Get("/templates", h.ListTemplates)
			r.Post("/templates", h.CreateTemplate)
			r.Get("/templates/{id}", h.GetTemplate)
			r.Put("/templates/{id}", h.UpdateTemplate)
			r.Delete("/templates/{id}", h.DeleteTemplate)
			r.Post("/templates/{id}/test", h.TestTemplate)

			// Webhooks.
			r.Get("/webhooks", h.ListWebhooks)
			r.Post("/webhooks", h.CreateWebhook)
			r.Delete("/webhooks/{id}", h.DeleteWebhook)
			r.Get("/webhooks/{id}/logs", h.WebhookLogs)

			// Dashboard analytics.
			r.Get("/dashboard/overview", h.DashboardOverview)
			r.Get("/dashboard/analytics", h.DashboardAnalytics)
			r.Get("/dashboard/live", h.DashboardLive)
		})

		// Routes that accept either an API key or a JWT (server-to-server +
		// dashboard). Workspace-scoped rate limit.
		r.Group(func(r chi.Router) {
			r.Use(mw.APIKeyOrJWT(h.Store, jwtSecret))
			r.Use(rl.ByWorkspace(1000, time.Minute))

			// Notifications.
			r.Post("/notifications/send", h.SendNotification)
			r.Post("/notifications/send-bulk", h.SendBulkNotifications)
			r.Get("/notifications", h.ListNotifications)
			r.Get("/notifications/{id}", h.GetNotification)
			r.Get("/notifications/{id}/deliveries", h.GetNotificationDeliveries)

			// Subscribers.
			r.Get("/subscribers", h.ListSubscribers)
			r.Post("/subscribers", h.UpsertSubscriber)
			r.Get("/subscribers/{externalId}", h.GetSubscriber)
			r.Put("/subscribers/{externalId}", h.UpdateSubscriber)
			r.Delete("/subscribers/{externalId}", h.DeleteSubscriber)
			r.Post("/subscribers/{externalId}/web-push", h.RegisterWebPush)

			// Subscriber WS token issuance.
			r.Post("/subscribers/ws-token", h.IssueSubscriberToken)
		})
	})

	return r
}
