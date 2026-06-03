package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/astrazstudio/pushnotify/server/pkg/crypto"
)

// APIKeyResolver resolves an API key hash to a workspace id. *store.Store and
// any test fake implement it.
type APIKeyResolver interface {
	ResolveAPIKey(ctx context.Context, keyHash string) (string, error)
}

// APIKeyOrJWT authenticates a request via either an X-API-Key header (resolving
// the workspace) or, failing that, a Bearer JWT. This lets server-to-server
// integrations use API keys while the dashboard uses JWTs.
func APIKeyOrJWT(st APIKeyResolver, jwtSecret string) func(http.Handler) http.Handler {
	jwtMW := JWTAuth(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := extractAPIKey(r)
			if apiKey != "" {
				workspaceID, err := st.ResolveAPIKey(r.Context(), crypto.HashToken(apiKey))
				if err != nil {
					writeUnauthorized(w, "invalid api key")
					return
				}
				ctx := WithWorkspace(r.Context(), workspaceID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Fall back to JWT auth.
			jwtMW(next).ServeHTTP(w, r)
		})
	}
}

func extractAPIKey(r *http.Request) string {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return strings.TrimSpace(k)
	}
	// Also accept "Authorization: Bearer pk_..." style API keys.
	if t := bearerToken(r); strings.HasPrefix(t, "pk_") {
		return t
	}
	return ""
}
