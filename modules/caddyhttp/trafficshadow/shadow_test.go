package trafficshadow

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
		Targets: []Target{
			{
				Name: "test-target",
				URL:  "http://shadow.example.com",
			},
		},
		SamplingRate:    1.0,
		CompareResponses: true,
	}
	h.Provision(caddy.ActiveContext())

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	err := h.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
		return nil
	}))

	if err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}

	// Check that main response is correct
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandler_captureRequest(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest("POST", "/api/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")

	captured := h.captureRequest(req)

	if captured.Method != "POST" {
		t.Errorf("Expected method POST, got %s", captured.Method)
	}

	if captured.URL != "/api/test" {
		t.Errorf("Expected URL /api/test, got %s", captured.URL)
	}

	if captured.Headers["X-API-Key"] != "test-key" {
		t.Errorf("Expected header X-API-Key=test-key, got %s", captured.Headers["X-API-Key"])
	}
}

func TestHandler_compareResponses(t *testing.T) {
	h := &Handler{}

	req := &CapturedRequest{ID: "test"}

	// Mock main response
	mainResp := &responseCapture{
		statusCode: http.StatusOK,
	}
	mainResp.body.WriteString(`{"status": "ok"}`)

	// Mock shadow response
	shadowResp := &http.Response{
		StatusCode: http.StatusOK,
	}
	shadowBody := []byte(`{"status": "ok"}`)

	comparison := h.compareResponses(req, mainResp, shadowResp, shadowBody, time.Millisecond, time.Millisecond, "test-target")

	if !comparison.Similar {
		t.Error("Responses should be similar")
	}

	if len(comparison.Differences) > 0 {
		t.Errorf("Expected no differences, got %v", comparison.Differences)
	}
}