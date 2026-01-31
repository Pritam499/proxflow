package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func TestHandler_ServeHTTP(t *testing.T) {
	h := &Handler{
		Rules: []Rule{
			{
				Algorithm: "token_bucket",
				Rate:      1,
				Burst:     1,
				Key:       "ip",
				Reject:    "429",
			},
		},
	}
	h.Provision(caddy.ActiveContext())

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// First request should pass
	err := h.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return nil
	}))
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Second request should be rate limited
	w2 := httptest.NewRecorder()
	err = h.ServeHTTP(w2, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return nil
	}))
	if err == nil {
		t.Error("Expected rate limit error")
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	tb := &TokenBucket{
		rate:       1, // 1 token per second
		burst:      2,
		tokens:     make(map[string]float64),
		lastRefill: make(map[string]time.Time),
	}

	key := "test"

	// First request should be allowed
	allowed, wait := tb.Allow(key)
	if !allowed {
		t.Error("First request should be allowed")
	}

	// Second request should be allowed (burst)
	allowed, wait = tb.Allow(key)
	if !allowed {
		t.Error("Second request should be allowed")
	}

	// Third request should be denied
	allowed, wait = tb.Allow(key)
	if allowed {
		t.Error("Third request should be denied")
	}
	if wait <= 0 {
		t.Error("Wait time should be positive")
	}
}

func TestSlidingWindow_Allow(t *testing.T) {
	sw := &SlidingWindow{
		rate:     2, // 2 requests per window
		window:   time.Minute,
		requests: make(map[string][]time.Time),
	}

	key := "test"

	// First request should be allowed
	allowed, _ := sw.Allow(key)
	if !allowed {
		t.Error("First request should be allowed")
	}

	// Second request should be allowed
	allowed, _ = sw.Allow(key)
	if !allowed {
		t.Error("Second request should be allowed")
	}

	// Third request should be denied
	allowed, wait := sw.Allow(key)
	if allowed {
		t.Error("Third request should be denied")
	}
	if wait <= 0 {
		t.Error("Wait time should be positive")
	}
}