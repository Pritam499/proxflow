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

package grpcgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/caddyconfig/httpcaddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp/reverseproxy"
)

func init() {
	caddy.RegisterModule(Gateway{})
	httpcaddyfile.RegisterHandlerDirective("grpc_gateway", parseGateway)
}

// Gateway implements a bidirectional gRPC gateway with protocol translation.
type Gateway struct {
	// Backend is the gRPC backend address.
	Backend string `json:"backend,omitempty"`

	// ProtoFiles contains the protobuf definitions.
	ProtoFiles []string `json:"proto_files,omitempty"`

	// Services maps service names to their configurations.
	Services map[string]*ServiceConfig `json:"services,omitempty"`

	// EnableReflection enables gRPC server reflection.
	EnableReflection bool `json:"enable_reflection,omitempty"`

	// EnableGRPCWeb enables gRPC-Web compatibility.
	EnableGRPCWeb bool `json:"enable_grpc_web,omitempty"`

	// AllowCORS allows CORS for web clients.
	AllowCORS bool `json:"allow_cors,omitempty"`

	// client connection pool
	clients map[string]*grpcClient
}

// ServiceConfig defines configuration for a gRPC service.
type ServiceConfig struct {
	// Methods maps HTTP methods to gRPC methods.
	Methods map[string]*MethodConfig `json:"methods,omitempty"`

	// Streaming indicates if this service supports streaming.
	Streaming bool `json:"streaming,omitempty"`
}

// MethodConfig defines configuration for a gRPC method.
type MethodConfig struct {
	// GRPCMethod is the full gRPC method name.
	GRPCMethod string `json:"grpc_method,omitempty"`

	// RequestMapping defines how to map HTTP request to gRPC.
	RequestMapping *Mapping `json:"request_mapping,omitempty"`

	// ResponseMapping defines how to map gRPC response to HTTP.
	ResponseMapping *Mapping `json:"response_mapping,omitempty"`
}

// Mapping defines field mappings for request/response transformation.
type Mapping struct {
	// FieldMappings maps HTTP fields to gRPC fields.
	FieldMappings map[string]string `json:"field_mappings,omitempty"`

	// BodyField specifies which gRPC field should contain the HTTP body.
	BodyField string `json:"body_field,omitempty"`
}

// grpcClient represents a connection to a gRPC service.
type grpcClient struct {
	// In a real implementation, this would hold gRPC client connections
	// For now, we'll use HTTP for simulation
	httpClient *http.Client
}

// CaddyModule returns the Caddy module information.
func (Gateway) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.grpc_gateway",
		New: func() caddy.Module { return new(Gateway) },
	}
}

// Provision sets up the gateway.
func (g *Gateway) Provision(ctx caddy.Context) error {
	if g.Backend == "" {
		return fmt.Errorf("backend is required")
	}

	g.clients = make(map[string]*grpcClient)

	// Initialize clients for each service
	for serviceName := range g.Services {
		g.clients[serviceName] = &grpcClient{
			httpClient: &http.Client{},
		}
	}

	return nil
}

// ServeHTTP handles HTTP requests and translates them to gRPC.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Check if this is a gRPC-Web request
	if g.isGRPCWebRequest(r) {
		return g.handleGRPCWeb(w, r)
	}

	// Extract service and method from URL path
	service, method, err := g.parsePath(r.URL.Path)
	if err != nil {
		return caddyhttp.Error(http.StatusNotFound, err)
	}

	// Get service config
	serviceConfig, exists := g.Services[service]
	if !exists {
		return caddyhttp.Error(http.StatusNotFound, fmt.Errorf("service not found: %s", service))
	}

	// Get method config
	methodConfig, exists := serviceConfig.Methods[r.Method]
	if !exists {
		return caddyhttp.Error(http.StatusMethodNotAllowed, fmt.Errorf("method not allowed: %s", r.Method))
	}

	// Handle CORS if enabled
	if g.AllowCORS && r.Method == http.MethodOptions {
		g.handleCORS(w, r)
		return nil
	}

	// Translate and forward the request
	return g.translateAndForward(w, r, service, methodConfig)
}

// isGRPCWebRequest checks if the request is gRPC-Web.
func (g *Gateway) isGRPCWebRequest(r *http.Request) bool {
	return g.EnableGRPCWeb &&
		(r.Header.Get("Content-Type") == "application/grpc-web" ||
			r.Header.Get("Content-Type") == "application/grpc-web-text")
}

// handleGRPCWeb handles gRPC-Web requests.
func (g *Gateway) handleGRPCWeb(w http.ResponseWriter, r *http.Request) {
	// Set appropriate headers for gRPC-Web
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type,x-grpc-web,x-user-agent")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	// In a real implementation, this would handle base64-encoded gRPC-Web protocol
	// For now, treat as regular request
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("gRPC-Web not fully implemented"))
}

// parsePath extracts service and method from the URL path.
func (g *Gateway) parsePath(path string) (service, method string, err error) {
	// Assume path format: /service/method
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid path format")
	}
	return parts[0], parts[1], nil
}

// handleCORS handles CORS preflight requests.
func (g *Gateway) handleCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
	w.WriteHeader(http.StatusOK)
}

// translateAndForward translates HTTP request to gRPC and forwards it.
func (g *Gateway) translateAndForward(w http.ResponseWriter, r *http.Request, service string, methodConfig *MethodConfig) error {
	client, exists := g.clients[service]
	if !exists {
		return fmt.Errorf("no client for service: %s", service)
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	// Parse JSON request
	var jsonRequest map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &jsonRequest); err != nil {
			return err
		}
	}

	// Apply request mapping
	grpcRequest := g.applyRequestMapping(jsonRequest, methodConfig.RequestMapping)

	// Convert to protobuf format (simplified)
	protoRequest, err := g.jsonToProto(grpcRequest)
	if err != nil {
		return err
	}

	// Create gRPC request
	grpcURL := fmt.Sprintf("%s/%s", g.Backend, methodConfig.GRPCMethod)

	grpcReq, err := http.NewRequest("POST", grpcURL, bytes.NewReader(protoRequest))
	if err != nil {
		return err
	}

	// Set gRPC headers
	grpcReq.Header.Set("Content-Type", "application/grpc+proto")
	grpcReq.Header.Set("grpc-accept-encoding", "gzip")

	// Forward to gRPC service
	resp, err := client.httpClient.Do(grpcReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read gRPC response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Convert from protobuf to JSON
	jsonResponse, err := g.protoToJSON(respBody)
	if err != nil {
		return err
	}

	// Apply response mapping
	httpResponse := g.applyResponseMapping(jsonResponse, methodConfig.ResponseMapping)

	// Write HTTP response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(httpResponse)
}

// applyRequestMapping applies request field mappings.
func (g *Gateway) applyRequestMapping(request map[string]interface{}, mapping *Mapping) map[string]interface{} {
	if mapping == nil {
		return request
	}

	result := make(map[string]interface{})

	// Apply field mappings
	for httpField, grpcField := range mapping.FieldMappings {
		if value, exists := request[httpField]; exists {
			result[grpcField] = value
		}
	}

	// Handle body field
	if mapping.BodyField != "" {
		result[mapping.BodyField] = request
	}

	return result
}

// applyResponseMapping applies response field mappings.
func (g *Gateway) applyResponseMapping(response map[string]interface{}, mapping *Mapping) map[string]interface{} {
	if mapping == nil {
		return response
	}

	result := make(map[string]interface{})

	// Apply field mappings (reverse direction)
	for grpcField, httpField := range mapping.FieldMappings {
		if value, exists := response[grpcField]; exists {
			result[httpField] = value
		}
	}

	return result
}

// jsonToProto converts JSON to protobuf format (simplified).
func (g *Gateway) jsonToProto(data map[string]interface{}) ([]byte, error) {
	// In a real implementation, this would use protobuf encoding
	// For now, just marshal to JSON as a placeholder
	return json.Marshal(data)
}

// protoToJSON converts protobuf to JSON format (simplified).
func (g *Gateway) protoToJSON(data []byte) (map[string]interface{}, error) {
	// In a real implementation, this would decode protobuf
	// For now, just unmarshal from JSON as a placeholder
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	return result, err
}

// UnmarshalCaddyfile parses the grpc_gateway directive.
func (g *Gateway) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	args := d.RemainingArgs()
	if len(args) != 1 {
		return d.ArgErr()
	}
	g.Backend = args[0]

	for d.NextBlock(0) {
		switch d.Val() {
		case "proto_file":
			args := d.RemainingArgs()
			g.ProtoFiles = append(g.ProtoFiles, args...)
		case "service":
			if !d.NextArg() {
				return d.ArgErr()
			}
			serviceName := d.Val()

			if g.Services == nil {
				g.Services = make(map[string]*ServiceConfig)
			}
			g.Services[serviceName] = &ServiceConfig{
				Methods: make(map[string]*MethodConfig),
			}

			service := g.Services[serviceName]
			for d.NextBlock(1) {
				switch d.Val() {
				case "method":
					if !d.NextArg() {
						return d.ArgErr()
					}
					httpMethod := d.Val()
					if !d.NextArg() {
						return d.ArgErr()
					}
					grpcMethod := d.Val()

					service.Methods[httpMethod] = &MethodConfig{
						GRPCMethod: grpcMethod,
					}
				case "streaming":
					service.Streaming = true
				}
			}
		case "enable_reflection":
			g.EnableReflection = true
		case "enable_grpc_web":
			g.EnableGRPCWeb = true
		case "allow_cors":
			g.AllowCORS = true
		default:
			return d.Errf("unrecognized option '%s'", d.Val())
		}
	}
	return nil
}

// parseGateway parses the grpc_gateway directive.
func parseGateway(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
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
	_ caddyfile.Unmarshaler       = (*Gateway)(nil)
)