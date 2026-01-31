package grpcgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func TestGateway_ServeHTTP(t *testing.T) {
	g := &Gateway{
		Backend: "http://grpc-backend:50051",
		Services: map[string]*ServiceConfig{
			"test": {
				Methods: map[string]*MethodConfig{
					"POST": {
						GRPCMethod: "TestService/TestMethod",
					},
				},
			},
		},
	}
	g.Provision(caddy.ActiveContext())

	req := httptest.NewRequest("POST", "/test/testmethod", strings.NewReader(`{"message": "hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Since we don't have a real gRPC backend, this will fail, but we can test the parsing
	err := g.ServeHTTP(w, req, caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return nil
	}))

	// Expect an error since backend is not reachable
	if err == nil {
		t.Error("Expected error due to unreachable backend")
	}
}

func TestGateway_parsePath(t *testing.T) {
	g := &Gateway{}

	tests := []struct {
		path     string
		service  string
		method   string
		hasError bool
	}{
		{"/user/get", "user", "get", false},
		{"/product/create", "product", "create", false},
		{"/invalid", "", "", true},
	}

	for _, tt := range tests {
		service, method, err := g.parsePath(tt.path)
		if (err != nil) != tt.hasError {
			t.Errorf("parsePath(%s) error = %v, expected error = %v", tt.path, err, tt.hasError)
		}
		if service != tt.service || method != tt.method {
			t.Errorf("parsePath(%s) = (%s, %s), expected (%s, %s)", tt.path, service, method, tt.service, tt.method)
		}
	}
}

func TestGateway_applyRequestMapping(t *testing.T) {
	g := &Gateway{}

	request := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
	}

	mapping := &Mapping{
		FieldMappings: map[string]string{
			"name":  "user_name",
			"email": "user_email",
		},
		BodyField: "data",
	}

	result := g.applyRequestMapping(request, mapping)

	if result["user_name"] != "John" {
		t.Errorf("Expected user_name = John, got %v", result["user_name"])
	}
	if result["user_email"] != "john@example.com" {
		t.Errorf("Expected user_email = john@example.com, got %v", result["user_email"])
	}
	if result["data"] == nil {
		t.Error("Expected data field to be set")
	}
}