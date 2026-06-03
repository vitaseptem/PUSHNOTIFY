package middleware

import (
	"net/http"
	"strings"

	"github.com/astrazstudio/pushnotify/server/internal/token"
)

// JWTAuth validates a Bearer JWT and injects the user/workspace identity.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				writeUnauthorized(w, "missing bearer token")
				return
			}
			claims, err := token.Parse(secret, raw)
			if err != nil {
				writeUnauthorized(w, "invalid token")
				return
			}
			ctx := WithIdentity(r.Context(), claims.UserID, claims.WorkspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
