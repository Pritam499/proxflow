package transform

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func TestHandler_ServeHTTP(t *testing.T) {
	h := &Handler{
		Rules: []Rule{
			{
				HeaderActions: []HeaderAction{
					{Action: "add", Name: "X-Test", Value: "value"},
				},
			},
		},
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	err := h.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return nil
	}))

	if err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}

	if w.Header().Get("X-Test") != "value" {
		t.Errorf("Expected header X-Test=value, got %s", w.Header().Get("X-Test"))
	}
}

func TestHandler_transformBody(t *testing.T) {
	h := &Handler{}

	body := `{"name": "test"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	transformed, err := h.transformBody(req, []BodyAction{
		{Action: "replace", Path: "name", Value: "transformed"},
	})

	if err != nil {
		t.Fatalf("transformBody failed: %v", err)
	}

	expected := `{"name":"transformed"}`
	if transformed != expected {
		t.Errorf("Expected %s, got %s", expected, transformed)
	}
}