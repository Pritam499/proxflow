package cache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func TestHandler_ServeHTTP(t *testing.T) {
	h := &Handler{}
	h.Provision(caddy.ActiveContext())

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	callCount := 0
	err := h.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return nil
	}))

	if err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected backend to be called once, got %d", callCount)
	}

	// Second request should be cached
	w2 := httptest.NewRecorder()
	err = h.ServeHTTP(w2, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return nil
	}))

	if err != nil {
		t.Fatalf("Second ServeHTTP failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected backend to not be called again, but was called %d times", callCount)
	}

	if w2.Header().Get("X-Cache") != "HIT" {
		t.Error("Expected X-Cache header to be HIT")
	}
}

func TestInMemoryCache(t *testing.T) {
	cache := &InMemoryCache{
		entries: make(map[string]*CacheEntry),
		maxSize: 1024 * 1024, // 1MB
	}

	entry := &CacheEntry{
		Key:   "test",
		Data:  []byte("test data"),
		TTL:   time.Hour,
	}

	cache.Set("test", entry)

	retrieved, found := cache.Get("test")
	if !found {
		t.Error("Expected to find cache entry")
	}

	if string(retrieved.Data) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(retrieved.Data))
	}

	cache.Delete("test")

	_, found = cache.Get("test")
	if found {
		t.Error("Expected entry to be deleted")
	}
}

func TestHandler_partialUpdate(t *testing.T) {
	h := &Handler{}
	h.Cache = &InMemoryCache{entries: make(map[string]*CacheEntry)}

	// Set initial cached JSON
	initialJSON := `{"user": {"name": "John", "age": 30}}`
	entry := &CacheEntry{
		Key:       "test",
		Data:      []byte(initialJSON),
		Timestamp: time.Now(),
		TTL:       time.Hour,
	}
	h.Cache.Set("test", entry)

	// Perform partial update
	update := &PartialUpdate{
		Key:   "test",
		Path:  "user.name",
		Value: "Jane",
	}
	h.partialUpdate(update)

	// Check updated data
	updated, found := h.Cache.Get("test")
	if !found {
		t.Fatal("Cache entry not found after update")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(updated.Data, &data); err != nil {
		t.Fatalf("Failed to unmarshal updated data: %v", err)
	}

	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatal("User field not found or not a map")
	}

	if user["name"] != "Jane" {
		t.Errorf("Expected name to be 'Jane', got '%v'", user["name"])
	}
}