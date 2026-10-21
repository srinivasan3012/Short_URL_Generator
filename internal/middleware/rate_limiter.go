package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientBucket struct {
	tokens     float64
	lastUpdate time.Time
}

// RateLimiter implements a thread-safe token-bucket rate limiter per client IP.
type RateLimiter struct {
	mu           sync.Mutex
	clients      map[string]*clientBucket
	rate         float64 // tokens per second
	burst        float64 // max token capacity
	cleanupTimer *time.Ticker
}

// NewRateLimiter initializes a rate limiter with given rate (RPS) and burst capacity.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	limiter := &RateLimiter{
		clients:      make(map[string]*clientBucket),
		rate:         rps,
		burst:        float64(burst),
		cleanupTimer: time.NewTicker(3 * time.Minute),
	}

	// Background routine to evict idle client records to prevent memory leaks
	go limiter.cleanupLoop()

	return limiter
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientBucket{
			tokens:     rl.burst - 1,
			lastUpdate: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.burst {
		b.tokens = rl.burst
	}
	b.lastUpdate = now

	if b.tokens >= 1 {
		b.tokens -= 1
		return true
	}

	return false
}

func (rl *RateLimiter) cleanupLoop() {
	for range rl.cleanupTimer.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.clients {
			if now.Sub(b.lastUpdate) > 5*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Stop terminates the cleanup background goroutine.
func (rl *RateLimiter) Stop() {
	if rl.cleanupTimer != nil {
		rl.cleanupTimer.Stop()
	}
}

// Middleware returns an HTTP middleware wrapping the rate limiter.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !rl.allow(ip) {
			reqID := GetRequestID(r.Context())
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)

			resp := map[string]interface{}{
				"error": map[string]interface{}{
					"code":       "RATE_LIMITED",
					"message":    "Rate limit exceeded. Please slow down your requests.",
					"request_id": reqID,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	// Respect X-Forwarded-For if behind a proxy
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
