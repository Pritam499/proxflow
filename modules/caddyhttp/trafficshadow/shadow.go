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

package trafficshadow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/caddyconfig/httpcaddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective("traffic_shadow", parseTrafficShadow)
}

// Handler implements traffic shadowing and replay for testing.
type Handler struct {
	// Targets is a list of shadow targets.
	Targets []Target `json:"targets,omitempty"`

	// SamplingRate is the rate of requests to shadow (0.0-1.0).
	SamplingRate float64 `json:"sampling_rate,omitempty"`

	// CompareResponses enables response comparison for regression detection.
	CompareResponses bool `json:"compare_responses,omitempty"`

	// LogDifferences logs response differences.
	LogDifferences bool `json:"log_differences,omitempty"`

	// Replay enables request replay functionality.
	Replay *ReplayConfig `json:"replay,omitempty"`
}

// Target defines a shadow target.
type Target struct {
	// Name is the target identifier.
	Name string `json:"name,omitempty"`

	// URL is the target endpoint.
	URL string `json:"url,omitempty"`

	// Headers to add to shadow requests.
	Headers map[string]string `json:"headers,omitempty"`

	// Timeout for shadow requests.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// IgnoreErrors prevents shadow request errors from affecting main response.
	IgnoreErrors bool `json:"ignore_errors,omitempty"`
}

// ReplayConfig configures request replay.
type ReplayConfig struct {
	// Storage configures where to store captured requests.
	Storage *StorageConfig `json:"storage,omitempty"`

	// ReplayRate is the rate at which to replay requests.
	ReplayRate float64 `json:"replay_rate,omitempty"`

	// MaxReplays is the maximum number of times to replay each request.
	MaxReplays int `json:"max_replays,omitempty"`
}

// StorageConfig configures request storage.
type StorageConfig struct {
	// Type can be "memory", "file", "redis".
	Type string `json:"type,omitempty"`

	// Path for file storage.
	Path string `json:"path,omitempty"`

	// Redis config for Redis storage.
	Redis *RedisConfig `json:"redis,omitempty"`
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

// CapturedRequest represents a captured request for replay.
type CapturedRequest struct {
	ID        string            `json:"id"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Timestamp time.Time         `json:"timestamp"`
	Replays   int               `json:"replays"`
}

// ResponseComparison represents a response comparison result.
type ResponseComparison struct {
	RequestID     string    `json:"request_id"`
	Target        string    `json:"target"`
	StatusCode    int       `json:"status_code"`
	ShadowCode    int       `json:"shadow_code"`
	ResponseTime  time.Duration `json:"response_time"`
	ShadowTime    time.Duration `json:"shadow_time"`
	Differences   []string  `json:"differences"`
	Similar       bool      `json:"similar"`
	Timestamp     time.Time `json:"timestamp"`
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.traffic_shadow",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the traffic shadow handler.
func (h *Handler) Provision(ctx caddy.Context) error {
	if h.SamplingRate == 0 {
		h.SamplingRate = 1.0 // Shadow all requests by default
	}
	if h.SamplingRate > 1.0 {
		h.SamplingRate = 1.0
	}
	if h.SamplingRate < 0 {
		h.SamplingRate = 0
	}

	for i := range h.Targets {
		if h.Targets[i].Timeout == 0 {
			h.Targets[i].Timeout = caddy.Duration(30 * time.Second)
		}
	}

	if h.Replay != nil {
		if h.Replay.ReplayRate == 0 {
			h.Replay.ReplayRate = 1.0
		}
		if h.Replay.MaxReplays == 0 {
			h.Replay.MaxReplays = 3
		}
	}

	return nil
}

// ServeHTTP handles traffic shadowing.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Capture the original request
	capturedReq := h.captureRequest(r)

	// Store for replay if configured
	if h.Replay != nil {
		h.storeRequest(capturedReq)
	}

	// Execute the main request
	mainStart := time.Now()
	capture := &responseCapture{ResponseWriter: w}
	err := next.ServeHTTP(capture, r)
	mainDuration := time.Since(mainStart)

	if err != nil {
		return err
	}

	// Shadow to targets if sampling allows
	if h.shouldShadow() {
		go h.shadowToTargets(capturedReq, capture, mainDuration)
	}

	return nil
}

// captureRequest captures the incoming request.
func (h *Handler) captureRequest(r *http.Request) *CapturedRequest {
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body)) // Restore body

	headers := make(map[string]string)
	for k, v := range r.Header {
		headers[k] = strings.Join(v, ",")
	}

	id := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s%s%d", r.Method, r.URL.String(), time.Now().UnixNano()))))

	return &CapturedRequest{
		ID:        id[:16], // Shorten hash
		Method:    r.Method,
		URL:       r.URL.String(),
		Headers:   headers,
		Body:      string(body),
		Timestamp: time.Now(),
		Replays:   0,
	}
}

// shouldShadow determines if this request should be shadowed.
func (h *Handler) shouldShadow() bool {
	// Simple random sampling - in real implementation, use proper sampling
	return true // For demo, shadow all
}

// shadowToTargets sends the request to shadow targets.
func (h *Handler) shadowToTargets(req *CapturedRequest, mainResponse *responseCapture, mainDuration time.Duration) {
	for _, target := range h.Targets {
		go h.shadowToTarget(req, target, mainResponse, mainDuration)
	}
}

// shadowToTarget sends the request to a specific target.
func (h *Handler) shadowToTarget(req *CapturedRequest, target Target, mainResponse *responseCapture, mainDuration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(target.Timeout))
	defer cancel()

	// Create shadow request
	shadowReq, err := http.NewRequestWithContext(ctx, req.Method, target.URL, strings.NewReader(req.Body))
	if err != nil {
		if !target.IgnoreErrors {
			// Log error
		}
		return
	}

	// Copy headers
	for k, v := range req.Headers {
		shadowReq.Header.Set(k, v)
	}

	// Add target-specific headers
	for k, v := range target.Headers {
		shadowReq.Header.Set(k, v)
	}

	// Execute shadow request
	shadowStart := time.Now()
	client := &http.Client{Timeout: time.Duration(target.Timeout)}
	shadowResp, err := client.Do(shadowReq)
	shadowDuration := time.Since(shadowStart)

	if err != nil {
		if !target.IgnoreErrors {
			// Log error
		}
		return
	}
	defer shadowResp.Body.Close()

	// Read shadow response
	shadowBody, err := io.ReadAll(shadowResp.Body)
	if err != nil {
		if !target.IgnoreErrors {
			// Log error
		}
		return
	}

	// Compare responses if enabled
	if h.CompareResponses {
		comparison := h.compareResponses(req, mainResponse, shadowResp, shadowBody, mainDuration, shadowDuration, target.Name)
		h.handleComparison(comparison)
	}
}

// compareResponses compares main and shadow responses.
func (h *Handler) compareResponses(req *CapturedRequest, mainResp *responseCapture, shadowResp *http.Response, shadowBody []byte, mainTime, shadowTime time.Duration, targetName string) *ResponseComparison {
	comparison := &ResponseComparison{
		RequestID:    req.ID,
		Target:       targetName,
		StatusCode:   mainResp.statusCode,
		ShadowCode:   shadowResp.StatusCode,
		ResponseTime: mainTime,
		ShadowTime:   shadowTime,
		Timestamp:    time.Now(),
		Similar:      true,
		Differences:  []string{},
	}

	// Compare status codes
	if mainResp.statusCode != shadowResp.StatusCode {
		comparison.Similar = false
		comparison.Differences = append(comparison.Differences,
			fmt.Sprintf("Status codes differ: %d vs %d", mainResp.statusCode, shadowResp.StatusCode))
	}

	// Compare response bodies (simplified JSON comparison)
	if !bytes.Equal(mainResp.body.Bytes(), shadowBody) {
		comparison.Similar = false
		comparison.Differences = append(comparison.Differences, "Response bodies differ")
	}

	// Check for significant timing differences
	if shadowTime > mainTime*2 {
		comparison.Differences = append(comparison.Differences,
			fmt.Sprintf("Shadow response significantly slower: %v vs %v", shadowTime, mainTime))
	}

	return comparison
}

// handleComparison processes the comparison result.
func (h *Handler) handleComparison(comparison *ResponseComparison) {
	if !comparison.Similar && h.LogDifferences {
		// In real implementation, log to configured logger or send to monitoring system
		fmt.Printf("Response difference detected: %+v\n", comparison)
	}

	// Could send to monitoring/alerting system
}

// storeRequest stores the request for potential replay.
func (h *Handler) storeRequest(req *CapturedRequest) {
	// In real implementation, store to configured storage (memory, file, Redis)
	// For now, just keep in memory with a simple map
}

// replayRequests replays stored requests to targets.
func (h *Handler) replayRequests() {
	// Implementation for replaying stored requests
	// This would run in a background goroutine
}

// UnmarshalCaddyfile parses the traffic_shadow directive.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	for d.NextBlock(0) {
		switch d.Val() {
		case "sampling_rate":
			if !d.NextArg() {
				return d.ArgErr()
			}
			rate, err := strconv.ParseFloat(d.Val(), 64)
			if err != nil {
				return d.Errf("invalid sampling_rate: %v", err)
			}
			h.SamplingRate = rate
		case "compare_responses":
			h.CompareResponses = true
		case "log_differences":
			h.LogDifferences = true
		case "target":
			target := Target{}
			if !d.NextArg() {
				return d.ArgErr()
			}
			target.Name = d.Val()
			if !d.NextArg() {
				return d.ArgErr()
			}
			target.URL = d.Val()

			for d.NextBlock(1) {
				switch d.Val() {
				case "header":
					args := d.RemainingArgs()
					if len(args) >= 2 {
						if target.Headers == nil {
							target.Headers = make(map[string]string)
						}
						target.Headers[args[0]] = args[1]
					}
				case "timeout":
					if !d.NextArg() {
						return d.ArgErr()
					}
					timeout, err := caddy.ParseDuration(d.Val())
					if err != nil {
						return d.Errf("invalid timeout: %v", err)
					}
					target.Timeout = caddy.Duration(timeout)
				case "ignore_errors":
					target.IgnoreErrors = true
				}
			}
			h.Targets = append(h.Targets, target)
		case "replay":
			replay := &ReplayConfig{}
			for d.NextBlock(1) {
				switch d.Val() {
				case "storage":
					storage := &StorageConfig{}
					for d.NextBlock(2) {
						switch d.Val() {
						case "type":
							if !d.NextArg() {
								return d.ArgErr()
							}
							storage.Type = d.Val()
						case "path":
							if !d.NextArg() {
								return d.ArgErr()
							}
							storage.Path = d.Val()
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
							storage.Redis = redis
						}
					}
					replay.Storage = storage
				case "replay_rate":
					if !d.NextArg() {
						return d.ArgErr()
					}
					rate, err := strconv.ParseFloat(d.Val(), 64)
					if err != nil {
						return d.Errf("invalid replay_rate: %v", err)
					}
					replay.ReplayRate = rate
				case "max_replays":
					if !d.NextArg() {
						return d.ArgErr()
					}
					max, err := strconv.Atoi(d.Val())
					if err != nil {
						return d.Errf("invalid max_replays: %v", err)
					}
					replay.MaxReplays = max
				}
			}
			h.Replay = replay
		default:
			return d.Errf("unrecognized option '%s'", d.Val())
		}
	}
	return nil
}

// parseTrafficShadow parses the traffic_shadow directive.
func parseTrafficShadow(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	ts := new(Handler)
	err := ts.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return ts, nil
}

// responseCapture captures the response for comparison.
type responseCapture struct {
	http.ResponseWriter
	body       bytes.Buffer
	headers    http.Header
	statusCode int
}

func (rc *responseCapture) Write(data []byte) (int, error) {
	rc.body.Write(data)
	return rc.ResponseWriter.Write(data)
}

func (rc *responseCapture) WriteHeader(statusCode int) {
	rc.statusCode = statusCode
	rc.headers = make(http.Header)
	for k, v := range rc.ResponseWriter.Header() {
		rc.headers[k] = v
	}
	rc.ResponseWriter.WriteHeader(statusCode)
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)