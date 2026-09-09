package middleware

import (
	"encoding/json"
	"fleettrack/internal/handler/dto"
	"fleettrack/internal/logger"
	"net"
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	kept := rl.requests[key][:0]
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= rl.limit {
		rl.requests[key] = kept
		return false
	}

	rl.requests[key] = append(kept, now)
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func RateLimit(limit int, window time.Duration, log logger.Logger) func(http.Handler) http.Handler {
	limiter := newRateLimiter(limit, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.allow(clientIP(r)) {
				respondTooManyRequests(w, r, log)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func respondTooManyRequests(w http.ResponseWriter, r *http.Request, log logger.Logger) {
	log.Warn("rate limit exceeded for " + clientIP(r))

	id, ok := r.Context().Value(RequestIDKey).(string)
	if !ok {
		id = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	if err := json.NewEncoder(w).Encode(dto.ErrorResponse{
		Status:    "error",
		Message:   "too many requests",
		RequestID: id,
	}); err != nil {
		log.Error(err.Error())
	}
}
