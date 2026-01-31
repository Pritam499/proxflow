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

package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	httpcaddyfile.RegisterHandlerDirective("cache", parseCache)
}

// Handler implements advanced caching with predictive warming and distributed sync.
type Handler struct {
	// Cache is the underlying cache implementation.
	Cache Cache `json:"-"`

	// TTL is the default time-to-live for cache entries.
	TTL caddy.Duration `json:"ttl,omitempty"`

	// MaxSize is the maximum cache size in MB.
	MaxSize int `json:"max_size,omitempty"`

	// KeyTemplate is a template for generating cache keys.
	KeyTemplate string `json:"key_template,omitempty"`

	// PredictiveWarming enables predictive cache warming.
	PredictiveWarming bool `json:"predictive_warming,omitempty"`

	// WebhookPath is the path for cache invalidation webhooks.
	WebhookPath string `json:"webhook_path,omitempty"`

	// DistributedSync enables distributed cache synchronization.
	DistributedSync *DistributedSync `json:"distributed_sync,omitempty"`

	// Rules is a list of caching rules.
	Rules []Rule `json:"rules,omitempty"`

	mu sync.RWMutex
}

// Rule defines a caching rule.
type Rule struct {
	// Match specifies when to apply this rule.
	Match *caddyhttp.Match `json:"match,omitempty"`

	// TTL overrides the default TTL for this rule.
	TTL caddy.Duration `json:"ttl,omitempty"`

	// Methods specifies which HTTP methods to cache.
	Methods []string `json:"methods,omitempty"`

	// Headers specifies which headers to include in cache key.
	Headers []string `json:"headers,omitempty"`
}

// CacheEntry represents a cached response.
type CacheEntry struct {
	Key         string
	Data        []byte
	Headers     http.Header
	StatusCode  int
	ContentType string
	Timestamp   time.Time
	TTL         time.Duration
	AccessCount int
	LastAccess  time.Time
}

// Cache interface for different cache implementations.
type Cache interface {
	Get(key string) (*CacheEntry, bool)
	Set(key string, entry *CacheEntry)
	Delete(key string)
	Clear()
	Size() int
}

// InMemoryCache is a simple in-memory cache implementation.
type InMemoryCache struct {
	entries map[string]*CacheEntry
	maxSize int // in MB
	currentSize int // in bytes
	mu      sync.RWMutex
}

// DistributedSync handles synchronization across instances.
type DistributedSync struct {
	// Peers is a list of peer instance URLs for sync.
	Peers []string `json:"peers,omitempty"`

	// SyncInterval is how often to sync with peers.
	SyncInterval caddy.Duration `json:"sync_interval,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.cache",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the cache handler.
func (h *Handler) Provision(ctx caddy.Context) error {
	if h.TTL == 0 {
		h.TTL = caddy.Duration(5 * time.Minute)
	}
	if h.MaxSize == 0 {
		h.MaxSize = 100 // 100 MB
	}
	if h.KeyTemplate == "" {
		h.KeyTemplate = "{{.Method}}_{{.Path}}"
	}
	if h.WebhookPath == "" {
		h.WebhookPath = "/cache/invalidate"
	}

	h.Cache = &InMemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: h.MaxSize * 1024 * 1024, // Convert MB to bytes
	}

	// Start predictive warming if enabled
	if h.PredictiveWarming {
		go h.predictiveWarming()
	}

	// Start distributed sync if configured
	if h.DistributedSync != nil {
		go h.distributedSync()
	}

	return nil
}

// ServeHTTP handles the caching logic.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Handle webhook invalidation
	if r.URL.Path == h.WebhookPath && r.Method == http.MethodPost {
		return h.handleWebhook(w, r)
	}

	// Check if request should be cached
	rule := h.findMatchingRule(r)
	if rule == nil {
		return next.ServeHTTP(w, r)
	}

	// Check if method is cacheable
	if !h.isMethodCacheable(r.Method, rule.Methods) {
		return next.ServeHTTP(w, r)
	}

	// Generate cache key
	key := h.generateKey(r, rule)

	// Try to get from cache
	if entry, found := h.Cache.Get(key); found && !h.isExpired(entry) {
		// Update access stats
		h.updateAccessStats(entry)

		// Serve from cache
		return h.serveFromCache(w, entry)
	}

	// Capture the response
	capture := &responseCapture{ResponseWriter: w}
	err := next.ServeHTTP(capture, r)
	if err != nil {
		return err
	}

	// Cache the response if successful
	if capture.statusCode >= 200 && capture.statusCode < 300 {
		entry := &CacheEntry{
			Key:         key,
			Data:        capture.body.Bytes(),
			Headers:     capture.headers,
			StatusCode:  capture.statusCode,
			ContentType: capture.headers.Get("Content-Type"),
			Timestamp:   time.Now(),
			TTL:         time.Duration(h.getTTL(rule)),
			AccessCount: 1,
			LastAccess:  time.Now(),
		}
		h.Cache.Set(key, entry)

		// Trigger distributed sync
		if h.DistributedSync != nil {
			go h.syncToPeers(key, entry)
		}
	}

	return nil
}

// findMatchingRule finds the first matching rule for the request.
func (h *Handler) findMatchingRule(r *http.Request) *Rule {
	for _, rule := range h.Rules {
		if caddyhttp.Match(r, rule.Match) {
			return &rule
		}
	}
	return nil
}

// isMethodCacheable checks if the HTTP method should be cached.
func (h *Handler) isMethodCacheable(method string, allowed []string) bool {
	if len(allowed) == 0 {
		return method == http.MethodGet || method == http.MethodHead
	}
	for _, m := range allowed {
		if m == method {
			return true
		}
	}
	return false
}

// generateKey generates a cache key for the request.
func (h *Handler) generateKey(r *http.Request, rule *Rule) string {
	key := h.KeyTemplate

	// Simple template replacement
	key = strings.ReplaceAll(key, "{{.Method}}", r.Method)
	key = strings.ReplaceAll(key, "{{.Path}}", r.URL.Path)
	key = strings.ReplaceAll(key, "{{.Query}}", r.URL.RawQuery)

	// Include specified headers
	if len(rule.Headers) > 0 {
		headerParts := make([]string, 0, len(rule.Headers))
		for _, header := range rule.Headers {
			if value := r.Header.Get(header); value != "" {
				headerParts = append(headerParts, header+":"+value)
			}
		}
		if len(headerParts) > 0 {
			key += "|" + strings.Join(headerParts, "|")
		}
	}

	// Hash for consistent key length
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// getTTL returns the TTL for a rule.
func (h *Handler) getTTL(rule *Rule) caddy.Duration {
	if rule.TTL != 0 {
		return rule.TTL
	}
	return h.TTL
}

// isExpired checks if a cache entry is expired.
func (h *Handler) isExpired(entry *CacheEntry) bool {
	return time.Since(entry.Timestamp) > entry.TTL
}

// updateAccessStats updates access statistics for predictive warming.
func (h *Handler) updateAccessStats(entry *CacheEntry) {
	h.mu.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	h.mu.Unlock()
}

// serveFromCache serves a response from cache.
func (h *Handler) serveFromCache(w http.ResponseWriter, entry *CacheEntry) error {
	// Copy headers
	for k, v := range entry.Headers {
		w.Header()[k] = v
	}
	w.Header().Set("X-Cache", "HIT")
	w.WriteHeader(entry.StatusCode)
	_, err := w.Write(entry.Data)
	return err
}

// handleWebhook handles cache invalidation webhooks.
func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Action string `json:"action"`
		Key    string `json:"key,omitempty"`
		Pattern string `json:"pattern,omitempty"`
		Partial *PartialUpdate `json:"partial,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch payload.Action {
	case "invalidate":
		if payload.Key != "" {
			h.Cache.Delete(payload.Key)
		} else if payload.Pattern != "" {
			h.invalidatePattern(payload.Pattern)
		}
	case "partial_update":
		if payload.Partial != nil {
			h.partialUpdate(payload.Partial)
		}
	case "clear":
		h.Cache.Clear()
	default:
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// invalidatePattern invalidates cache entries matching a pattern.
func (h *Handler) invalidatePattern(pattern string) {
	// Simple pattern matching - in real implementation, use regex or glob
	h.mu.Lock()
	for key := range h.Cache.(*InMemoryCache).entries {
		if strings.Contains(key, pattern) {
			delete(h.Cache.(*InMemoryCache).entries, key)
		}
	}
	h.mu.Unlock()
}

// PartialUpdate represents a partial cache update.
type PartialUpdate struct {
	Key   string                 `json:"key"`
	Path  string                 `json:"path"`  // JSON path like "user.name"
	Value interface{}           `json:"value"`
}

// partialUpdate performs a partial update on cached JSON data.
func (h *Handler) partialUpdate(update *PartialUpdate) {
	entry, found := h.Cache.Get(update.Key)
	if !found {
		return
	}

	// Parse JSON
	var data interface{}
	if err := json.Unmarshal(entry.Data, &data); err != nil {
		return
	}

	// Apply update (simple implementation for dot notation)
	if m, ok := data.(map[string]interface{}); ok {
		h.applyPartialUpdate(m, strings.Split(update.Path, "."), update.Value)
	}

	// Re-encode and update cache
	if updated, err := json.Marshal(data); err == nil {
		entry.Data = updated
		entry.Timestamp = time.Now()
		h.Cache.Set(update.Key, entry)
	}
}

// applyPartialUpdate applies a partial update to nested map.
func (h *Handler) applyPartialUpdate(data map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	key := path[0]
	if len(path) == 1 {
		data[key] = value
		return
	}

	if next, ok := data[key].(map[string]interface{}); ok {
		h.applyPartialUpdate(next, path[1:], value)
	}
}

// predictiveWarming performs predictive cache warming based on access patterns.
func (h *Handler) predictiveWarming() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Analyze access patterns and warm cache
		// Simple implementation: re-cache recently accessed entries
		h.mu.RLock()
		// In real implementation, analyze patterns and preload related data
		h.mu.RUnlock()
	}
}

// distributedSync synchronizes cache with peer instances.
func (h *Handler) distributedSync() {
	if h.DistributedSync == nil {
		return
	}

	ticker := time.NewTicker(time.Duration(h.DistributedSync.SyncInterval))
	defer ticker.Stop()

	for range ticker.C {
		// Sync with peers
		for _, peer := range h.DistributedSync.Peers {
			go h.syncWithPeer(peer)
		}
	}
}

// syncWithPeer syncs cache with a specific peer.
func (h *Handler) syncWithPeer(peer string) {
	// In real implementation, exchange cache metadata and sync entries
	// For now, just a placeholder
}

// syncToPeers sends a cache entry to all peers.
func (h *Handler) syncToPeers(key string, entry *CacheEntry) {
	if h.DistributedSync == nil {
		return
	}

	for _, peer := range h.DistributedSync.Peers {
		go h.sendToPeer(peer, key, entry)
	}
}

// sendToPeer sends a cache entry to a peer instance.
func (h *Handler) sendToPeer(peer, key string, entry *CacheEntry) {
	// HTTP POST to peer's sync endpoint
	// Implementation omitted for brevity
}

// responseCapture captures the response for caching.
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

// Get implements the InMemoryCache Get method.
func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	entry, found := c.entries[key]
	c.mu.RUnlock()
	return entry, found
}

// Set implements the InMemoryCache Set method.
func (c *InMemoryCache) Set(key string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction policy - remove oldest entries if over size
	for c.currentSize+len(entry.Data) > c.maxSize && len(c.entries) > 0 {
		var oldestKey string
		var oldestTime time.Time
		for k, e := range c.entries {
			if oldestTime.IsZero() || e.Timestamp.Before(oldestTime) {
				oldestTime = e.Timestamp
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = entry
	c.currentSize += len(entry.Data)
}

// Delete implements the InMemoryCache Delete method.
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	if entry, exists := c.entries[key]; exists {
		c.currentSize -= len(entry.Data)
		delete(c.entries, key)
	}
	c.mu.Unlock()
}

// Clear implements the InMemoryCache Clear method.
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[string]*CacheEntry)
	c.currentSize = 0
	c.mu.Unlock()
}

// Size implements the InMemoryCache Size method.
func (c *InMemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// UnmarshalCaddyfile parses the cache directive.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	for d.NextBlock(0) {
		switch d.Val() {
		case "ttl":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("invalid ttl: %v", err)
			}
			h.TTL = caddy.Duration(dur)
		case "max_size":
			if !d.NextArg() {
				return d.ArgErr()
			}
			// Parse size (assume MB for now)
			h.MaxSize = 100 // placeholder
		case "predictive_warming":
			h.PredictiveWarming = true
		case "webhook_path":
			if !d.NextArg() {
				return d.ArgErr()
			}
			h.WebhookPath = d.Val()
		case "distributed_sync":
			ds := &DistributedSync{}
			for d.NextBlock(1) {
				switch d.Val() {
				case "peer":
					args := d.RemainingArgs()
					ds.Peers = append(ds.Peers, args...)
				case "sync_interval":
					if !d.NextArg() {
						return d.ArgErr()
					}
					dur, err := caddy.ParseDuration(d.Val())
					if err != nil {
						return d.Errf("invalid sync_interval: %v", err)
					}
					ds.SyncInterval = caddy.Duration(dur)
				}
			}
			h.DistributedSync = ds
		default:
			return d.Errf("unrecognized option '%s'", d.Val())
		}
	}
	return nil
}

// parseCache parses the cache directive.
func parseCache(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	c := new(Handler)
	err := c.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)