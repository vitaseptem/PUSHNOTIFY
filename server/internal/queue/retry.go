package queue

import (
	"math"
	"math/rand"
	"time"
)

const maxBackoff = time.Hour

// CalculateBackoff returns an exponential backoff with jitter for a retry
// attempt. base*2^attempt seconds plus up to 20% random jitter, capped at 1h.
//
//	attempt 1 -> ~30s, attempt 2 -> ~60s, attempt 3 -> ~120s (base=30)
func CalculateBackoff(attempt int, baseSeconds int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	secs := float64(baseSeconds) * math.Pow(2, float64(attempt-1))
	jitter := secs * 0.2 * rand.Float64()
	d := time.Duration((secs + jitter) * float64(time.Second))
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

// ShouldRetry reports whether another attempt is permitted for the job.
func ShouldRetry(job QueueJob) bool {
	return job.Attempt < job.MaxAttempts
}
