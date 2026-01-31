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

package tenants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(App{})
}

// App manages multi-tenant configurations with dynamic routing.
type App struct {
	// Tenants is a map of tenant configurations.
	Tenants map[string]*Tenant `json:"tenants,omitempty"`

	// APIEndpoint is the endpoint for tenant management API.
	APIEndpoint string `json:"api_endpoint,omitempty"`

	// DefaultTenant is the default tenant configuration to inherit from.
	DefaultTenant *Tenant `json:"default_tenant,omitempty"`

	mu sync.RWMutex
}

// Tenant represents a tenant configuration.
type Tenant struct {
	// Name is the tenant identifier.
	Name string `json:"name,omitempty"`

	// Domains is a list of domains for this tenant.
	Domains []string `json:"domains,omitempty"`

	// Routes is the HTTP routes for this tenant.
	Routes caddyhttp.RouteList `json:"routes,omitempty"`

	// TLS configuration for this tenant.
	TLS *caddyhttp.TLSConfig `json:"tls,omitempty"`

	// RateLimit configuration.
	RateLimit *RateLimit `json:"rate_limit,omitempty"`

	// Metrics configuration.
	Metrics *Metrics `json:"metrics,omitempty"`

	// Inherits specifies the parent tenant to inherit from.
	Inherits string `json:"inherits,omitempty"`
}

// RateLimit defines rate limiting for a tenant.
type RateLimit struct {
	// Requests per second.
	RPS int `json:"rps,omitempty"`

	// Burst size.
	Burst int `json:"burst,omitempty"`
}

// Metrics defines metrics configuration for a tenant.
type Metrics struct {
	// Enabled specifies if metrics are enabled for this tenant.
	Enabled bool `json:"enabled,omitempty"`

	// Prefix for metric names.
	Prefix string `json:"prefix,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "tenants",
		New: func() caddy.Module { return new(App) },
	}
}

// Provision sets up the app.
func (a *App) Provision(ctx caddy.Context) error {
	if a.APIEndpoint == "" {
		a.APIEndpoint = "/api/tenants"
	}
	if a.Tenants == nil {
		a.Tenants = make(map[string]*Tenant)
	}
	return nil
}

// Start starts the tenants app.
func (a *App) Start() error {
	// Register API endpoints
	http.HandleFunc(a.APIEndpoint, a.handleAPI)
	http.HandleFunc(a.APIEndpoint+"/", a.handleTenantAPI)
	return nil
}

// Stop stops the tenants app.
func (a *App) Stop() error {
	return nil
}

// handleAPI handles tenant management API requests.
func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listTenants(w, r)
	case http.MethodPost:
		a.createTenant(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTenantAPI handles individual tenant API requests.
func (a *App) handleTenantAPI(w http.ResponseWriter, r *http.Request) {
	// Extract tenant name from URL
	tenantName := r.URL.Path[len(a.APIEndpoint)+1:]
	if tenantName == "" {
		http.Error(w, "Tenant name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		a.getTenant(w, r, tenantName)
	case http.MethodPut:
		a.updateTenant(w, r, tenantName)
	case http.MethodDelete:
		a.deleteTenant(w, r, tenantName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listTenants returns all tenants.
func (a *App) listTenants(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.Tenants)
}

// createTenant creates a new tenant.
func (a *App) createTenant(w http.ResponseWriter, r *http.Request) {
	var tenant Tenant
	if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	if _, exists := a.Tenants[tenant.Name]; exists {
		a.mu.Unlock()
		http.Error(w, "Tenant already exists", http.StatusConflict)
		return
	}

	// Apply inheritance
	if tenant.Inherits != "" {
		if parent, ok := a.Tenants[tenant.Inherits]; ok {
			a.inheritFromParent(&tenant, parent)
		} else if a.DefaultTenant != nil {
			a.inheritFromParent(&tenant, a.DefaultTenant)
		}
	} else if a.DefaultTenant != nil {
		a.inheritFromParent(&tenant, a.DefaultTenant)
	}

	a.Tenants[tenant.Name] = &tenant
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tenant)
}

// inheritFromParent applies parent configuration to tenant.
func (a *App) inheritFromParent(tenant, parent *Tenant) {
	if tenant.Routes == nil {
		tenant.Routes = parent.Routes
	}
	if tenant.TLS == nil {
		tenant.TLS = parent.TLS
	}
	if tenant.RateLimit == nil {
		tenant.RateLimit = parent.RateLimit
	}
	if tenant.Metrics == nil {
		tenant.Metrics = parent.Metrics
	}
	if len(tenant.Domains) == 0 {
		tenant.Domains = parent.Domains
	}
}

// getTenant returns a specific tenant.
func (a *App) getTenant(w http.ResponseWriter, r *http.Request, name string) {
	a.mu.RLock()
	tenant, exists := a.Tenants[name]
	a.mu.RUnlock()

	if !exists {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

// updateTenant updates a tenant.
func (a *App) updateTenant(w http.ResponseWriter, r *http.Request, name string) {
	var updates Tenant
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	tenant, exists := a.Tenants[name]
	if !exists {
		a.mu.Unlock()
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	// Apply updates
	if updates.Domains != nil {
		tenant.Domains = updates.Domains
	}
	if updates.Routes != nil {
		tenant.Routes = updates.Routes
	}
	if updates.TLS != nil {
		tenant.TLS = updates.TLS
	}
	if updates.RateLimit != nil {
		tenant.RateLimit = updates.RateLimit
	}
	if updates.Metrics != nil {
		tenant.Metrics = updates.Metrics
	}

	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

// deleteTenant deletes a tenant.
func (a *App) deleteTenant(w http.ResponseWriter, r *http.Request, name string) {
	a.mu.Lock()
	_, exists := a.Tenants[name]
	if !exists {
		a.mu.Unlock()
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}

	delete(a.Tenants, name)
	a.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// GetTenantRoutes returns routes for a tenant based on the request.
func (a *App) GetTenantRoutes(r *http.Request) (caddyhttp.RouteList, *Tenant) {
	host := r.Host

	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, tenant := range a.Tenants {
		for _, domain := range tenant.Domains {
			if domain == host || (domain == "*" && host != "") {
				return tenant.Routes, tenant
			}
		}
	}

	// Return default if no tenant matches
	if a.DefaultTenant != nil {
		return a.DefaultTenant.Routes, a.DefaultTenant
	}

	return nil, nil
}

// Interface guards
var (
	_ caddy.App    = (*App)(nil)
	_ caddy.Module = (*App)(nil)
)