package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a fixed-window rate limiter backed by Redis.
type RateLimiter struct {
	rdb *redis.Client
}

// NewRateLimiter constructs a RateLimiter.
func NewRateLimiter(rdb *redis.Client) *RateLimiter {
	return &RateLimiter{rdb: rdb}
}

// ByIP limits requests per client IP (used on public routes).
func (rl *RateLimiter) ByIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return rl.limit(func(r *http.Request) string {
		return "ip:" + clientIP(r)
	}, limit, window)
}

// ByWorkspace limits requests per authenticated workspace.
func (rl *RateLimiter) ByWorkspace(limit int, window time.Duration) func(http.Handler) http.Handler {
	return rl.limit(func(r *http.Request) string {
		ws := WorkspaceID(r.Context())
		if ws == "" {
			ws = clientIP(r)
		}
		return "ws:" + ws
	}, limit, window)
}

func (rl *RateLimiter) limit(keyFn func(*http.Request) string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bucket := time.Now().Truncate(window).Unix()
			key := fmt.Sprintf("pushnotify:ratelimit:%s:%d", keyFn(r), bucket)

			count, err := rl.rdb.Incr(r.Context(), key).Result()
			if err != nil {
				// Fail open: never let Redis hiccups break the API.
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				_ = rl.rdb.Expire(r.Context(), key, window).Err()
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			remaining := limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if int(count) > limit {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// First IP in the list is the original client.
		if i := indexComma(xff); i >= 0 {
			return xff[:i]
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}
