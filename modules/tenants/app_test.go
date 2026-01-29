package tenants

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Pritam499/proxflow/v2"
)

func TestApp_Provision(t *testing.T) {
	a := &App{}
	ctx := caddy.ActiveContext()
	err := a.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if a.APIEndpoint != "/api/tenants" {
		t.Errorf("Expected APIEndpoint /api/tenants, got %s", a.APIEndpoint)
	}
}

func TestApp_createTenant(t *testing.T) {
	a := &App{}
	a.Provision(caddy.ActiveContext())

	tenant := Tenant{
		Name:    "test-tenant",
		Domains: []string{"test.com"},
	}

	body, _ := json.Marshal(tenant)
	req := httptest.NewRequest("POST", "/api/tenants", bytes.NewReader(body))
	w := httptest.NewRecorder()

	a.handleAPI(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	if len(a.Tenants) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(a.Tenants))
	}
}

func TestApp_GetTenantRoutes(t *testing.T) {
	a := &App{
		Tenants: map[string]*Tenant{
			"test": {
				Name:    "test",
				Domains: []string{"test.com"},
			},
		},
	}

	req := httptest.NewRequest("GET", "http://test.com/path", nil)
	routes, tenant := a.GetTenantRoutes(req)

	if tenant == nil || tenant.Name != "test" {
		t.Error("Expected to find test tenant")
	}

	if routes != nil {
		t.Error("Expected nil routes for test tenant")
	}
}