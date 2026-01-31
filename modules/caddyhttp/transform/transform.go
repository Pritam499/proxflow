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

package transform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/caddyconfig/httpcaddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Handler{})
	httpcaddyfile.RegisterHandlerDirective("transform", parseTransform)
}

// Handler implements request/response transformation for API gateway functionality.
type Handler struct {
	// Rules is a list of transformation rules to apply.
	Rules []Rule `json:"rules,omitempty"`
}

// Rule defines a transformation rule.
type Rule struct {
	// Match specifies when to apply this rule.
	Match *caddyhttp.Match `json:"match,omitempty"`

	// HeaderActions is a list of header manipulations.
	HeaderActions []HeaderAction `json:"header_actions,omitempty"`

	// BodyActions is a list of body transformations.
	BodyActions []BodyAction `json:"body_actions,omitempty"`

	// Enrichment specifies request enrichment.
	Enrichment *Enrichment `json:"enrichment,omitempty"`

	// Aggregation specifies response aggregation from multiple backends.
	Aggregation *Aggregation `json:"aggregation,omitempty"`
}

// HeaderAction defines a header manipulation.
type HeaderAction struct {
	// Action can be "add", "set", "delete".
	Action string `json:"action,omitempty"`

	// Name is the header name.
	Name string `json:"name,omitempty"`

	// Value is the header value (for add/set).
	Value string `json:"value,omitempty"`
}

// BodyAction defines a body transformation.
type BodyAction struct {
	// Action can be "replace", "append", "prepend".
	Action string `json:"action,omitempty"`

	// Path is the JSON path for transformation (simple dot notation).
	Path string `json:"path,omitempty"`

	// Value is the value to set.
	Value interface{} `json:"value,omitempty"`

	// Template is a Go template for dynamic values.
	Template string `json:"template,omitempty"`
}

// Enrichment defines request enrichment.
type Enrichment struct {
	// Data is key-value pairs to add to the request.
	Data map[string]interface{} `json:"data,omitempty"`

	// FromURL specifies a URL to fetch additional data from.
	FromURL string `json:"from_url,omitempty"`

	// Method for the enrichment request.
	Method string `json:"method,omitempty"`

	// Headers for the enrichment request.
	Headers map[string]string `json:"headers,omitempty"`
}

// Aggregation defines response aggregation.
type Aggregation struct {
	// Backends is a list of backend URLs to aggregate from.
	Backends []string `json:"backends,omitempty"`

	// Method for aggregation requests.
	Method string `json:"method,omitempty"`

	// Combine specifies how to combine responses ("merge", "concat", "first").
	Combine string `json:"combine,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.transform",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the handler.
func (h *Handler) Provision(ctx caddy.Context) error {
	for i := range h.Rules {
		if h.Rules[i].Match == nil {
			h.Rules[i].Match = &caddyhttp.Match{}
		}
	}
	return nil
}

// ServeHTTP handles the request transformation.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// Apply transformation rules
	for _, rule := range h.Rules {
		if !caddyhttp.Match(r, rule.Match) {
			continue
		}

		// Apply header actions
		for _, action := range rule.HeaderActions {
			h.applyHeaderAction(r, action)
		}

		// Apply body actions
		if len(rule.BodyActions) > 0 {
			body, err := h.transformBody(r, rule.BodyActions)
			if err != nil {
				return caddyhttp.Error(http.StatusInternalServerError, err)
			}
			r.Body = io.NopCloser(strings.NewReader(body))
			r.ContentLength = int64(len(body))
		}

		// Apply enrichment
		if rule.Enrichment != nil {
			err := h.enrichRequest(r, rule.Enrichment)
			if err != nil {
				return caddyhttp.Error(http.StatusInternalServerError, err)
			}
		}

		// Handle aggregation
		if rule.Aggregation != nil {
			return h.aggregateResponse(w, r, rule.Aggregation)
		}
	}

	return next.ServeHTTP(w, r)
}

// applyHeaderAction applies a header manipulation.
func (h *Handler) applyHeaderAction(r *http.Request, action HeaderAction) {
	switch action.Action {
	case "add":
		r.Header.Add(action.Name, action.Value)
	case "set":
		r.Header.Set(action.Name, action.Value)
	case "delete":
		r.Header.Del(action.Name)
	}
}

// transformBody applies body transformations.
func (h *Handler) transformBody(r *http.Request, actions []BodyAction) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body.Close()

	var data interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &data); err != nil {
			return string(body), nil // Not JSON, return as is
		}
	}

	for _, action := range actions {
		data = h.applyBodyAction(data, action)
	}

	result, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// applyBodyAction applies a single body action.
func (h *Handler) applyBodyAction(data interface{}, action BodyAction) interface{} {
	if data == nil {
		data = make(map[string]interface{})
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	keys := strings.Split(action.Path, ".")
	current := m

	for i, key := range keys {
		if i == len(keys)-1 {
			switch action.Action {
			case "replace":
				current[key] = action.Value
			case "append":
				if arr, ok := current[key].([]interface{}); ok {
					current[key] = append(arr, action.Value)
				}
			case "prepend":
				if arr, ok := current[key].([]interface{}); ok {
					current[key] = append([]interface{}{action.Value}, arr...)
				}
			}
		} else {
			if current[key] == nil {
				current[key] = make(map[string]interface{})
			}
			if next, ok := current[key].(map[string]interface{}); ok {
				current = next
			} else {
				break
			}
		}
	}

	return m
}

// enrichRequest adds data to the request.
func (h *Handler) enrichRequest(r *http.Request, enrichment *Enrichment) error {
	// Add static data
	for k, v := range enrichment.Data {
		r.Header.Set("X-Enriched-"+k, fmt.Sprintf("%v", v))
	}

	// Fetch from URL if specified
	if enrichment.FromURL != "" {
		method := enrichment.Method
		if method == "" {
			method = "GET"
		}

		req, err := http.NewRequest(method, enrichment.FromURL, nil)
		if err != nil {
			return err
		}

		for k, v := range enrichment.Headers {
			req.Header.Set(k, v)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		r.Header.Set("X-Enriched-Data", string(body))
	}

	return nil
}

// aggregateResponse aggregates responses from multiple backends.
func (h *Handler) aggregateResponse(w http.ResponseWriter, r *http.Request, agg *Aggregation) error {
	method := agg.Method
	if method == "" {
		method = r.Method
	}

	results := make([]interface{}, 0, len(agg.Backends))

	for _, backend := range agg.Backends {
		req, err := http.NewRequest(method, backend, r.Body)
		if err != nil {
			continue
		}

		// Copy headers
		for k, v := range r.Header {
			req.Header[k] = v
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			results = append(results, data)
		}
	}

	var combined interface{}
	switch agg.Combine {
	case "merge":
		combined = h.mergeResults(results)
	case "concat":
		combined = results
	case "first":
		if len(results) > 0 {
			combined = results[0]
		}
	default:
		combined = results
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(combined)
}

// mergeResults merges multiple JSON objects.
func (h *Handler) mergeResults(results []interface{}) interface{} {
	merged := make(map[string]interface{})
	for _, result := range results {
		if m, ok := result.(map[string]interface{}); ok {
			for k, v := range m {
				merged[k] = v
			}
		}
	}
	return merged
}

// UnmarshalCaddyfile parses the transform directive.
func (h *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume handler name

	for d.NextBlock(0) {
		rule := Rule{}

		// Parse matchers
		matcherSet, err := httpcaddyfile.ParseSegmentAsConfig(httpcaddyfile.Helper{Dispenser: d})
		if err != nil {
			return err
		}
		if len(matcherSet) > 0 {
			rule.Match = &caddyhttp.Match{}
			// Simplified: assume first matcher
			if route, ok := matcherSet[0].Value.(caddyhttp.Route); ok && len(route.MatcherSetsRaw) > 0 {
				rule.Match = &caddyhttp.Match{}
			}
		}

		// Parse subdirectives
		for d.NextBlock(1) {
			switch d.Val() {
			case "header":
				action := HeaderAction{}
				args := d.RemainingArgs()
				if len(args) >= 2 {
					action.Action = args[0]
					action.Name = args[1]
					if len(args) >= 3 {
						action.Value = args[2]
					}
				}
				rule.HeaderActions = append(rule.HeaderActions, action)
			case "body":
				action := BodyAction{}
				args := d.RemainingArgs()
				if len(args) >= 3 {
					action.Action = args[0]
					action.Path = args[1]
					action.Value = args[2]
				}
				rule.BodyActions = append(rule.BodyActions, action)
			case "enrich":
				enrich := &Enrichment{Data: make(map[string]interface{})}
				for d.NextBlock(2) {
					switch d.Val() {
					case "data":
						args := d.RemainingArgs()
						if len(args) >= 2 {
							enrich.Data[args[0]] = args[1]
						}
					case "from_url":
						if !d.NextArg() {
							return d.ArgErr()
						}
						enrich.FromURL = d.Val()
					}
				}
				rule.Enrichment = enrich
			case "aggregate":
				agg := &Aggregation{}
				for d.NextBlock(2) {
					switch d.Val() {
					case "backend":
						args := d.RemainingArgs()
						agg.Backends = append(agg.Backends, args...)
					case "combine":
						if !d.NextArg() {
							return d.ArgErr()
						}
						agg.Combine = d.Val()
					}
				}
				rule.Aggregation = agg
			}
		}

		h.Rules = append(h.Rules, rule)
	}
	return nil
}

// parseTransform parses the transform directive.
func parseTransform(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	t := new(Handler)
	err := t.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddyfile.Unmarshaler       = (*Handler)(nil)
)