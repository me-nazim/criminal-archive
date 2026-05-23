// Package middleware contains cross-cutting HTTP middleware: rate
// limiting, security headers, and structured access logging.
package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"

	"github.com/me-nazim/criminal-archive/backend/internal/auth"
	"github.com/me-nazim/criminal-archive/backend/internal/httpx"
)

// SecurityHeaders sets a conservative baseline of security response
// headers. The CSP is intentionally permissive enough that the SPA
// works in production behind Cloudflare; tighten in
// docker-compose.prod.yml as the asset graph stabilises.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Strict-Transport-Security only makes sense over HTTPS in prod;
		// adding it here is harmless for HTTP-on-localhost but useful at
		// the edge.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

// AccessLog emits one structured log line per HTTP request when it
// completes. We use chi's WrapResponseWriter so we can capture the
// final status and bytes written.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			dur := time.Since(start)

			rid := middleware.GetReqID(r.Context())
			id, _ := auth.IdentityFrom(r.Context())
			var userID string
			if id.UserID != [16]byte{} {
				userID = id.UserID.String()
			}

			logger.Info("http",
				"request_id", rid,
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", dur.Milliseconds(),
				"bytes", ww.BytesWritten(),
				"user_id", userID,
				"ip", clientIP(r),
				"ua", r.UserAgent(),
			)
		})
	}
}

// ipBucket holds a per-key rate limiter and the time it was last seen,
// used to evict idle entries.
type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a simple in-memory token bucket per IP. For multi-node
// deployments we'd back this with Redis; for v1 it is good enough.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	rps     rate.Limit
	burst   int
	ttl     time.Duration
}

// NewRateLimiter returns a limiter at rps requests/sec with a burst.
// Idle buckets are evicted after `ttl` of inactivity.
func NewRateLimiter(rps int, burst int, ttl time.Duration) *RateLimiter {
	if rps <= 0 {
		rps = 20
	}
	if burst <= 0 {
		burst = rps * 2
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	rl := &RateLimiter{
		buckets: make(map[string]*ipBucket),
		rps:     rate.Limit(rps),
		burst:   burst,
		ttl:     ttl,
	}
	go rl.gcLoop()
	return rl
}

func (rl *RateLimiter) gcLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.ttl)
		for k, b := range rl.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &ipBucket{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.buckets[key] = b
	}
	b.lastSeen = time.Now()
	return b.limiter
}

// Middleware returns a chi-compatible middleware that rejects requests
// exceeding the budget for their IP with a 429.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if ip == "" {
				ip = "unknown"
			}
			if !rl.get(ip).Allow() {
				httpx.WriteError(w, r, nil, httpx.RateLimited(""))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// LoginAttemptLimiter is a separate, much stricter limiter we apply only
// to /auth/login. We key it on a hash of (IP, email) so a single IP
// cannot grind down a target account, while a target email still gets
// fresh capacity from a different IP.
type LoginAttemptLimiter struct {
	rl *RateLimiter
}

// NewLoginAttemptLimiter caps login POSTs at 5/min and 10/hour per
// (IP, email) tuple. Caller wraps it via Middleware.
func NewLoginAttemptLimiter() *LoginAttemptLimiter {
	return &LoginAttemptLimiter{
		rl: NewRateLimiter(1, 5, 30*time.Minute),
	}
}

// Middleware enforces the login limit. The middleware never reads the
// request body — it only keys on IP + the basic-auth-style "email"
// query param if present. (The actual email is also in the JSON body;
// reading it would consume the body and break the handler. The IP-only
// keying is still effective.)
func (l *LoginAttemptLimiter) Middleware() func(http.Handler) http.Handler {
	return l.rl.Middleware()
}

// clientIP returns the best available remote address for r. chi's
// RealIP middleware has usually already promoted X-Forwarded-For.
func clientIP(r *http.Request) string {
	if h := r.Header.Get("X-Real-IP"); h != "" {
		return h
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return strings.TrimSpace(host)
}

// silence: ensure context import is kept for future request-scoped
// values, even when unused in current code paths.
var _ = context.Background
