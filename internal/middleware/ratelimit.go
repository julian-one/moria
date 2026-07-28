package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// ipLimiter tracks a single IP's rate limiter and the last time it was seen.
type ipLimiter struct {
	limiter  *rate.Limiter // The actual token-bucket rate limiter for the IP
	lastSeen atomic.Int64  // Unix timestamp of the last time this IP made a request
}

// rateLimiter manages a collection of per-IP token buckets.
type rateLimiter struct {
	rate     rate.Limit
	burst    int
	mu       sync.RWMutex
	limiters map[string]*ipLimiter
}

// NewRateLimiter returns a per-IP token-bucket middleware. r is the steady-state
// refill rate and burst is the maximum tokens that can accumulate. Idle IPs are
// evicted after one hour to bound memory.
func NewRateLimiter(ctx context.Context, r rate.Limit, burst int) Middleware {
	rl := &rateLimiter{
		rate:     r,
		burst:    burst,
		limiters: make(map[string]*ipLimiter),
	}

	// Launch background cleanup goroutine
	go rl.cleanup(ctx)

	// Return the middleware function
	return rl.middleware
}

// cleanup periodically removes stale IPs to prevent unbounded memory growth.
func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Unix()

			rl.mu.Lock()
			for ip, l := range rl.limiters {
				// Evict IPs that haven't been seen in the last hour
				if now-l.lastSeen.Load() > int64(time.Hour.Seconds()) {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// get lazily retrieves or initializes a limiter for a given IP.
func (rl *rateLimiter) get(ip string) *rate.Limiter {
	// Fast path: read-lock to check if the limiter already exists
	rl.mu.RLock()
	l, ok := rl.limiters[ip]
	rl.mu.RUnlock()

	if !ok {
		// Slow path: acquire write-lock and double-check
		rl.mu.Lock()
		l, ok = rl.limiters[ip] // Double-checked locking
		if !ok {
			l = &ipLimiter{limiter: rate.NewLimiter(rl.rate, rl.burst)}
			rl.limiters[ip] = l
		}
		rl.mu.Unlock()
	}

	// Update the last seen timestamp (atomic operation, safe without mutex)
	l.lastSeen.Store(time.Now().Unix())
	return l.limiter
}

// middleware enforces the rate limit on incoming HTTP requests.
func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Check if the current request IP is allowed to proceed
		if !rl.get(GetClientIP(req)).Allow() {
			// Rate limit exceeded; return a 429 Too Many Requests JSON error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests"})
			return
		}

		// Allow the request to proceed to the next handler
		next.ServeHTTP(w, req)
	})
}
