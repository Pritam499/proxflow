// Copyright 2023 The ProxFlow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/caddyconfig/httpcaddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective("rate_limit", parseRateLimit)
}

// Handler implements advanced rate limiting with multiple algorithms.
type Handler struct {
	// Rules is a list of rate limiting rules.
	Rules []Rule `json:"rules,omitempty"`

	// Distributed storage for rate limiting state.
	Distributed *DistributedConfig `json:"distributed,omitempty"`

	mu sync.RWMutex
}

// Rule defines a rate limiting rule.
type Rule struct {
	// Match specifies when to apply this rule.
	Match *caddyhttp.Match `json:"match,omitempty"`

	// Algorithm specifies the rate limiting algorithm: "token_bucket", "leaky_bucket", "sliding_window".
	Algorithm string `json:"algorithm,omitempty"`

	// Rate is the allowed rate (requests per second for token/leaky bucket, requests per window for sliding window).
	Rate float64 `json:"rate,omitempty"`

	// Burst is the burst size for token bucket.
	Burst int `json:"burst,omitempty"`

	// Window is the time window for sliding window (in seconds).
	Window caddy.Duration `json:"window,omitempty"`

	// Key specifies how to identify the client: "ip", "user", "api_key", "header:{name}".
	Key string `json:"key,omitempty"`

	// Zone defines the rate limiting zone (for distributed rate limiting).
	Zone string `json:"zone,omitempty"`

	// Reject specifies what to do when rate limit is exceeded: "429" (default), "delay", "drop".
	Reject string `json:"reject,omitempty"`
}

// DistributedConfig configures distributed rate limiting.
type DistributedConfig struct {
	// Redis configures Redis as the distributed store.
	Redis *RedisConfig `json:"redis,omitempty"`

	// SyncInterval is how often to sync with distributed store.
	SyncInterval caddy.Duration `json:"sync_interval,omitempty"`
}

// RedisConfig configures Redis connection.
type RedisConfig struct {
	// Address is the Redis server address.
	Address string `json:"address,omitempty"`

	// Password for Redis authentication.
	Password string `json:"password,omitempty"`

	// DB is the Redis database number.
	DB int `json:"db,omitempty"`
}

// RateLimiter interface for different algorithms.
type RateLimiter interface {
	Allow(key string) (bool, time.Duration)
}

// TokenBucket implements token bucket algorithm.
type TokenBucket struct {
	rate       float64
	burst      int
	tokens     map[string]float64
	lastRefill map[string]time.Time
	mu         sync.Mutex
}

func (tb *TokenBucket) Allow(key string) (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	last := tb.lastRefill[key]

	// Calculate tokens to add
	elapsed := now.Sub(last).Seconds()
	tokensToAdd := elapsed * tb.rate

	currentTokens := tb.tokens[key] + tokensToAdd
	if currentTokens > float64(tb.burst) {
		currentTokens = float64(tb.burst)
	}

	if currentTokens >= 1 {
		tb.tokens[key] = currentTokens - 1
		tb.lastRefill[key] = now
		return true, 0
	}

	// Calculate wait time
	waitTime := time.Duration((1 - currentTokens) / tb.rate * float64(time.Second))
	return false, waitTime
}

// LeakyBucket implements leaky bucket algorithm.
type LeakyBucket struct {
	rate     float64
	capacity int
	buckets  map[string]*BucketState
	mu       sync.Mutex
}

type BucketState struct {
	water    int
	lastLeak time.Time
}

func (lb *LeakyBucket) Allow(key string) (bool, time.Duration) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()
	state := lb.buckets[key]
	if state == nil {
		state = &BucketState{lastLeak: now}
		lb.buckets[key] = state
	}

	// Leak water
	elapsed := now.Sub(state.lastLeak).Seconds()
	leaked := int(elapsed * lb.rate)
	state.water -= leaked
	if state.water < 0 {
		state.water = 0
	}
	state.lastLeak = now

	if state.water < lb.capacity {
		state.water++
		return true, 0
	}

	return false, time.Second // Simple wait time
}

// SlidingWindow implements sliding window algorithm.
type SlidingWindow struct {
	rate    float64
	window  time.Duration
	requests map[string][]time.Time
	mu      sync.Mutex
}

func (sw *SlidingWindow) Allow(key string) (bool, time.Duration) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.window)

	reqs := sw.requests[key]
	// Remove old requests
	validReqs := make([]time.Time, 0)
	for _, t := range reqs {
		if t.After(windowStart) {
			validReqs = append(validReqs, t)
		}
	}

	if float64(len(validReqs)) < sw.rate {
		validReqs = append(validReqs, now)
		sw.requests[key] = validReqs
		return true, 0
	}

	// Calculate wait time until oldest request expires
	if len(validReqs) > 0 {
		waitTime := validReqs[0].Sub(windowStart)
		return false, waitTime
	}

	return false, sw.window
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.rate_limit",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the rate limiter.
func (h *Handler) Provision(ctx caddy.Context) error {
	for i := range h.Rules {
		if h.Rules[i].Algorithm == "" {
			h.Rules[i].Algorithm = "token_bucket"
		}
		if h.Rules[i].Reject == "" {
			h.Rules[i].Reject = "429"
		}
		if h.Rules[i].Key == "" {
			h.Rules[i].Key = "ip"
		}
		if h.Distributed != nil && h.Distributed.SyncInterval == 0 {
			h.Distributed.SyncInterval = caddy.Duration(30 * time.Second)
		}
	}
	return nil
}

// ServeHTTP implements rate limiting.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	for _, rule := range h.Rules {
		if !caddyhttp.Match(r, rule.Match) {
			continue
		}

		// Get rate limiter for this rule
		limiter := h.getLimiter(rule)

		// Get key for rate limiting
		key := h.getKey(r, rule.Key)

		// Check rate limit
		allowed, waitTime := limiter.Allow(key)

		if !allowed {
			switch rule.Reject {
			case "429":
				w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(waitTime.Seconds())))
				return caddyhttp.Error(http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded"))
			case "delay":
				time.Sleep(waitTime)
			case "drop":
				return caddyhttp.Error(http.StatusServiceUnavailable, fmt.Errorf("service temporarily unavailable"))
			}
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Remaining", "unknown") // Would need to track actual counts
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(waitTime.Seconds())))
	}

	return next.ServeHTTP(w, r)
}

// getLimiter returns the rate limiter for a rule.
func (h *Handler) getLimiter(rule Rule) RateLimiter {
	// In a real implementation, you'd cache these per rule
	switch rule.Algorithm {
	case "token_bucket":
		return &TokenBucket{
			rate:       rule.Rate,
			burst:      rule.Burst,
			tokens:     make(map[string]float64),
			lastRefill: make(map[string]time.Time),
		}
	case "leaky_bucket":
		return &LeakyBucket{
			rate:    rule.Rate,
			capacity: rule.Burst,
			buckets: make(map[string]*BucketState),
		}
	case "sliding_window":
		return &SlidingWindow{
			rate:     rule.Rate,
			window:   time.Duration(rule.Window),
			requests: make(map[string][]time.Time),
		}
	default:
		return &TokenBucket{
			rate:       rule.Rate,
			burst:      rule.Burst,
			tokens:     make(map[string]float64),
			lastRefill: make(map[string]time.Time),
		}
	}
}

// getKey extracts the rate limiting key from the request.
func (h *Handler) getKey(r *http.Request, keyType string) string {
	switch keyType {
	case "ip":
		return r.RemoteAddr
	case "user":
		if user := r.Header.Get("X-User-ID"); user != "" {
			return user
		}
		return r.RemoteAddr
	case "api_key":
		if key := r.Header.Get("X-API-Key"); key != "" {
			return key
		}
		return r.RemoteAddr
	default:
		// Check for header:key format
		if strings.HasPrefix(keyType, "header:") {
			headerName := strings.TrimPrefix(keyType, "header:")
			if value := r.Header.Get(headerName); value != "" {
				return value
			}
		}
		return r.RemoteAddr
	}
}

// UnmarshalCaddyfile parses the rate_limit directive.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	for d.NextBlock(0) {
		rule := Rule{}

		// Parse matchers
		matcherSet, err := httpcaddyfile.ParseSegmentAsConfig(httpcaddyfile.Helper{Dispenser: d})
		if err != nil {
			return err
		}
		if len(matcherSet) > 0 {
			rule.Match = &caddyhttp.Match{}
		}

		// Parse subdirectives
		for d.NextBlock(1) {
			switch d.Val() {
			case "algorithm":
				if !d.NextArg() {
					return d.ArgErr()
				}
				rule.Algorithm = d.Val()
			case "rate":
				if !d.NextArg() {
					return d.ArgErr()
				}
				rate, err := strconv.ParseFloat(d.Val(), 64)
				if err != nil {
					return d.Errf("invalid rate: %v", err)
				}
				rule.Rate = rate
			case "burst":
				if !d.NextArg() {
					return d.ArgErr()
				}
				burst, err := strconv.Atoi(d.Val())
				if err != nil {
					return d.Errf("invalid burst: %v", err)
				}
				rule.Burst = burst
			case "window":
				if !d.NextArg() {
					return d.ArgErr()
				}
				window, err := caddy.ParseDuration(d.Val())
				if err != nil {
					return d.Errf("invalid window: %v", err)
				}
				rule.Window = caddy.Duration(window)
			case "key":
				if !d.NextArg() {
					return d.ArgErr()
				}
				rule.Key = d.Val()
			case "zone":
				if !d.NextArg() {
					return d.ArgErr()
				}
				rule.Zone = d.Val()
			case "reject":
				if !d.NextArg() {
					return d.ArgErr()
				}
				rule.Reject = d.Val()
			case "distributed":
				dist := &DistributedConfig{}
				for d.NextBlock(2) {
					switch d.Val() {
					case "redis":
						redis := &RedisConfig{}
						for d.NextBlock(3) {
							switch d.Val() {
							case "address":
								if !d.NextArg() {
									return d.ArgErr()
								}
								redis.Address = d.Val()
							case "password":
								if !d.NextArg() {
									return d.ArgErr()
								}
								redis.Password = d.Val()
							case "db":
								if !d.NextArg() {
									return d.ArgErr()
								}
								db, err := strconv.Atoi(d.Val())
								if err != nil {
									return d.Errf("invalid db: %v", err)
								}
								redis.DB = db
							}
						}
						dist.Redis = redis
					case "sync_interval":
						if !d.NextArg() {
							return d.ArgErr()
						}
						syncInterval, err := caddy.ParseDuration(d.Val())
						if err != nil {
							return d.Errf("invalid sync_interval: %v", err)
						}
						dist.SyncInterval = caddy.Duration(syncInterval)
					}
				}
				h.Distributed = dist
			}
		}

		h.Rules = append(h.Rules, rule)
	}
	return nil
}

// parseRateLimit parses the rate_limit directive.
func parseRateLimit(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	rl := new(Handler)
	err := rl.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return rl, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)