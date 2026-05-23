package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestSecurityHeadersAreSet(t *testing.T) {
	t.Parallel()
	h := SecurityHeaders(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	for _, key := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Strict-Transport-Security",
	} {
		if rr.Header().Get(key) == "" {
			t.Errorf("expected %q to be set", key)
		}
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	t.Parallel()
	// 1 rps, burst=2: first 2 requests pass, third is blocked.
	rl := NewRateLimiter(1, 2, time.Minute)
	mw := rl.Middleware()(okHandler())

	pass := 0
	limited := 0
	for i := 0; i < 5; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:1000"
		mw.ServeHTTP(rr, req)
		if rr.Code == http.StatusOK {
			pass++
		} else if rr.Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if pass < 2 || limited < 1 {
		t.Fatalf("expected at least 2 pass and 1 limited, got pass=%d limited=%d", pass, limited)
	}
}

func TestRateLimiter_SeparateIPsHaveSeparateBuckets(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 1, time.Minute)
	mw := rl.Middleware()(okHandler())

	// Saturate IP A.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1000"
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first call should pass; got %d", rr.Code)
	}
	// Second call from same IP: limited.
	rr = httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second call from same IP should be limited; got %d", rr.Code)
	}
	// Different IP gets fresh capacity.
	rr = httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:1000"
	mw.ServeHTTP(rr, req2)
	if rr.Code != http.StatusOK {
		t.Errorf("different IP should pass; got %d", rr.Code)
	}
}

func TestAccessLogEmitsOneLinePerRequest(t *testing.T) {
	t.Parallel()
	// Capture into a buffer-backed logger.
	var mu sync.Mutex
	buf := &threadSafeBuffer{}
	h := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	mw := AccessLog(logger)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	mw.ServeHTTP(rr, req)

	mu.Lock()
	out := buf.String()
	mu.Unlock()
	if !strings.Contains(out, "msg=http") {
		t.Errorf("expected access log line; got %q", out)
	}
	if !strings.Contains(out, "path=/x") {
		t.Errorf("expected path label in log; got %q", out)
	}
}

// threadSafeBuffer is a minimal io.Writer guarded by a mutex.
type threadSafeBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *threadSafeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *threadSafeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// silence unused import in case io isn't otherwise needed
var _ = io.Discard
