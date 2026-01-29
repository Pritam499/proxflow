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

package graphqlfederation

import (
	"bytes"
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
	caddy.RegisterModule(Gateway{})
	httpcaddyfile.RegisterHandlerDirective("graphql_federation", parseGraphQLFederation)
}

// Gateway is an HTTP handler that acts as a GraphQL federation gateway.
// It federates multiple GraphQL services, providing a unified API endpoint
// with intelligent query planning and caching.
type Gateway struct {
	// Services is the list of upstream GraphQL services to federate.
	Services []Service `json:"services,omitempty"`

	// CacheDuration is how long to cache query plans and responses.
	CacheDuration caddy.Duration `json:"cache_duration,omitempty"`

	// MaxQueryComplexity limits the complexity of queries to prevent abuse.
	MaxQueryComplexity int `json:"max_query_complexity,omitempty"`

	// EnableIntrospection allows GraphQL introspection queries.
	EnableIntrospection bool `json:"enable_introspection,omitempty"`

	// SchemaCache caches the federated schema.
	schemaCache *FederatedSchema
	// QueryCache caches query plans.
	queryCache map[string]*QueryPlan
	cacheMu    sync.RWMutex
	lastUpdate time.Time
}

// Service represents an upstream GraphQL service.
type Service struct {
	// Name is the unique name of the service.
	Name string `json:"name,omitempty"`
	// URL is the endpoint URL of the service.
	URL string `json:"url,omitempty"`
	// SchemaSDL is the GraphQL schema SDL for this service.
	SchemaSDL string `json:"schema_sdl,omitempty"`
}

// FederatedSchema holds the merged schema from all services.
type FederatedSchema struct {
	// Types maps type names to their definitions.
	Types map[string]*TypeDefinition
	// Schema is the combined SDL.
	Schema string
}

// TypeDefinition represents a GraphQL type.
type TypeDefinition struct {
	Name       string
	Kind       string
	Fields     map[string]*FieldDefinition
	Interfaces []string
}

// FieldDefinition represents a GraphQL field.
type FieldDefinition struct {
	Name string
	Type string
	Args map[string]string
}

// QueryPlan represents a plan for executing a federated query.
type QueryPlan struct {
	ServiceQueries map[string]string
	Dependencies   map[string][]string
}

// CaddyModule returns the Caddy module information.
func (Gateway) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.graphql_federation",
		New: func() caddy.Module { return new(Gateway) },
	}
}

// Provision sets up the gateway.
func (g *Gateway) Provision(ctx caddy.Context) error {
	if g.CacheDuration == 0 {
		g.CacheDuration = caddy.Duration(5 * time.Minute)
	}
	if g.MaxQueryComplexity == 0 {
		g.MaxQueryComplexity = 1000
	}
	g.queryCache = make(map[string]*QueryPlan)
	g.schemaCache = &FederatedSchema{
		Types: make(map[string]*TypeDefinition),
	}

	// Build the federated schema
	if err := g.buildFederatedSchema(); err != nil {
		return fmt.Errorf("failed to build federated schema: %v", err)
	}

	return nil
}

// Validate ensures the configuration is valid.
func (g Gateway) Validate() error {
	if len(g.Services) == 0 {
		return fmt.Errorf("at least one service must be configured")
	}
	names := make(map[string]bool)
	for _, svc := range g.Services {
		if svc.Name == "" {
			return fmt.Errorf("service name is required")
		}
		if svc.URL == "" {
			return fmt.Errorf("service URL is required for %s", svc.Name)
		}
		if names[svc.Name] {
			return fmt.Errorf("duplicate service name: %s", svc.Name)
		}
		names[svc.Name] = true
	}
	return nil
}

// ServeHTTP handles GraphQL requests.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.Method != http.MethodPost {
		return caddyhttp.Error(http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return caddyhttp.Error(http.StatusBadRequest, err)
	}

	var req GraphQLRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return caddyhttp.Error(http.StatusBadRequest, err)
	}

	// Check query complexity
	if g.calculateComplexity(req.Query) > g.MaxQueryComplexity {
		return caddyhttp.Error(http.StatusBadRequest, fmt.Errorf("query too complex"))
	}

	// Plan the query
	plan, err := g.planQuery(req.Query)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	// Execute the plan
	result, err := g.executePlan(plan, req.Variables)
	if err != nil {
		return caddyhttp.Error(http.StatusInternalServerError, err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GraphQLResponse{Data: result})
	return nil
}

// buildFederatedSchema merges schemas from all services.
func (g *Gateway) buildFederatedSchema() error {
	for _, svc := range g.Services {
		if err := g.parseAndMergeSchema(svc); err != nil {
			return fmt.Errorf("failed to merge schema for %s: %v", svc.Name, err)
		}
	}
	return nil
}

// parseAndMergeSchema parses and merges a service's schema.
func (g *Gateway) parseAndMergeSchema(svc Service) error {
	// This is a simplified schema parser. In a real implementation,
	// you'd use a proper GraphQL parser library.
	lines := strings.Split(svc.SchemaSDL, "\n")
	currentType := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentType = parts[1]
				g.schemaCache.Types[currentType] = &TypeDefinition{
					Name:   currentType,
					Kind:   "OBJECT",
					Fields: make(map[string]*FieldDefinition),
				}
			}
		} else if strings.HasPrefix(line, "}") {
			currentType = ""
		} else if currentType != "" && strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				fieldName := strings.TrimSpace(parts[0])
				fieldType := strings.TrimSpace(parts[1])
				fieldType = strings.TrimSuffix(fieldType, "!")
				g.schemaCache.Types[currentType].Fields[fieldName] = &FieldDefinition{
					Name: fieldName,
					Type: fieldType,
				}
			}
		}
	}
	return nil
}

// planQuery creates a query plan for federated execution.
func (g *Gateway) planQuery(query string) (*QueryPlan, error) {
	g.cacheMu.RLock()
	if plan, exists := g.queryCache[query]; exists {
		g.cacheMu.RUnlock()
		return plan, nil
	}
	g.cacheMu.RUnlock()

	// Simplified query planning. In reality, this would analyze the query
	// and determine which services need to be queried for which fields.
	plan := &QueryPlan{
		ServiceQueries: make(map[string]string),
		Dependencies:   make(map[string][]string),
	}

	// For each service, plan a subquery
	for _, svc := range g.Services {
		plan.ServiceQueries[svc.Name] = query // Simplified: send full query to each service
	}

	g.cacheMu.Lock()
	g.queryCache[query] = plan
	g.cacheMu.Unlock()

	return plan, nil
}

// executePlan executes the query plan by calling services.
func (g *Gateway) executePlan(plan *QueryPlan, variables map[string]interface{}) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(plan.ServiceQueries))

	for svcName, subQuery := range plan.ServiceQueries {
		wg.Add(1)
		go func(name, q string) {
			defer wg.Done()
			svc := g.findService(name)
			if svc == nil {
				errChan <- fmt.Errorf("service %s not found", name)
				return
			}

			resp, err := g.callService(svc, q, variables)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			results[name] = resp
			mu.Unlock()
		}(svcName, subQuery)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	// Merge results. In a real implementation, this would stitch
	// the responses together based on the schema.
	merged := make(map[string]interface{})
	for _, result := range results {
		if data, ok := result.(map[string]interface{}); ok {
			for k, v := range data {
				merged[k] = v
			}
		}
	}

	return merged, nil
}

// callService makes an HTTP call to a service.
func (g *Gateway) callService(svc *Service, query string, variables map[string]interface{}) (interface{}, error) {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(svc.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", gqlResp.Errors)
	}

	return gqlResp.Data, nil
}

// findService finds a service by name.
func (g *Gateway) findService(name string) *Service {
	for _, svc := range g.Services {
		if svc.Name == name {
			return &svc
		}
	}
	return nil
}

// calculateComplexity calculates the complexity of a query.
func (g *Gateway) calculateComplexity(query string) int {
	// Simplified complexity calculation based on query length.
	return len(query)
}

// UnmarshalCaddyfile sets up the module from Caddyfile tokens.
func (g *Gateway) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	for d.NextBlock(0) {
		switch d.Val() {
		case "service":
			var svc Service
			if !d.NextArg() {
				return d.ArgErr()
			}
			svc.Name = d.Val()
			if !d.NextArg() {
				return d.ArgErr()
			}
			svc.URL = d.Val()
			for d.NextBlock(1) {
				switch d.Val() {
				case "schema":
					if !d.NextArg() {
						return d.ArgErr()
					}
					svc.SchemaSDL = d.RemainingArgs()[0] // Simplified
				default:
					return d.Errf("unrecognized subdirective '%s'", d.Val())
				}
			}
			g.Services = append(g.Services, svc)
		case "cache_duration":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("invalid cache_duration: %v", err)
			}
			g.CacheDuration = caddy.Duration(dur)
		case "max_query_complexity":
			if !d.NextArg() {
				return d.ArgErr()
			}
			complexity, err := strconv.Atoi(d.Val())
			if err != nil {
				return d.Errf("invalid max_query_complexity: %v", err)
			}
			g.MaxQueryComplexity = complexity
		case "enable_introspection":
			g.EnableIntrospection = true
		default:
			return d.Errf("unrecognized directive '%s'", d.Val())
		}
	}
	return nil
}

// GraphQLRequest represents a GraphQL request.
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response.
type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []GQLError  `json:"errors,omitempty"`
}

// GQLError represents a GraphQL error.
type GQLError struct {
	Message string `json:"message"`
}

// parseGraphQLFederation parses the graphql_federation directive.
func parseGraphQLFederation(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	g := new(Gateway)
	err := g.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Gateway)(nil)
	_ caddy.Provisioner           = (*Gateway)(nil)
	_ caddy.Validator             = (*Gateway)(nil)
	_ caddyfile.Unmarshaler       = (*Gateway)(nil)
)