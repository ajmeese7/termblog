package server

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Tracks connection attempts per IP using a sliding window
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	done     chan struct{}
	stopOnce sync.Once
}

// Creates a rate limiter with the given limit and time window
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		done:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Stops the cleanup goroutine. Call this when shutting down.
// Safe to call multiple times.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.done)
	})
}

// Checks if a connection from the given IP should be allowed.
// Returns whether the request is allowed and the current request count for the IP.
func (rl *RateLimiter) Allow(ip string) (allowed bool, count int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Filter to only recent requests
	var recent []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	// Check if over limit
	if len(recent) >= rl.limit {
		rl.requests[ip] = recent
		return false, len(recent)
	}

	// Record this request
	rl.requests[ip] = append(recent, now)
	return true, len(recent) + 1
}

// Periodically removes old entries to prevent memory leaks
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)

			for ip, times := range rl.requests {
				var recent []time.Time
				for _, t := range times {
					if t.After(cutoff) {
						recent = append(recent, t)
					}
				}

				if len(recent) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = recent
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Creates an HTTP middleware that enforces rate limits per IP
func HTTPRateLimitMiddleware(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		allowed, count := limiter.Allow(ip)
		if !allowed {
			log.Printf("Rate limited HTTP request from %s (%d requests in window, limit %d)", ip, count, limiter.limit)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Creates a Wish middleware that enforces rate limits
func RateLimitMiddleware(limiter *RateLimiter) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			// Extract IP from remote address
			ip, _, err := net.SplitHostPort(sess.RemoteAddr().String())
			if err != nil {
				// Can't parse IP, allow through
				next(sess)
				return
			}

			allowed, count := limiter.Allow(ip)
			if !allowed {
				log.Printf("Rate limited SSH connection from %s (%d requests in window, limit %d)", ip, count, limiter.limit)
				wish.Println(sess, "Rate limit exceeded. Please try again later.")
				return
			}

			next(sess)
		}
	}
}
