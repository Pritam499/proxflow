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

package trafficdashboard

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/caddyconfig/httpcaddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Dashboard{})
	httpcaddyfile.RegisterHandlerDirective("traffic_dashboard", parseDashboard)
}

//go:embed static/*
var staticFS embed.FS

// Dashboard is an HTTP handler that serves a real-time traffic visualization dashboard.
type Dashboard struct {
	// Path is the path where the dashboard is served. Default is "/dashboard".
	Path string `json:"path,omitempty"`

	// UpdateInterval is how often to send updates via SSE. Default is 1s.
	UpdateInterval caddy.Duration `json:"update_interval,omitempty"`
}

// TrafficData represents the data sent to clients.
type TrafficData struct {
	Timestamp    time.Time         `json:"timestamp"`
	RequestRate  float64           `json:"request_rate"`
	ErrorRate    float64           `json:"error_rate"`
	Latency      map[string]float64 `json:"latency"`
	ActiveConns  int               `json:"active_conns"`
	GeoData      map[string]int    `json:"geo_data"`
}

// CaddyModule returns the Caddy module information.
func (Dashboard) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.traffic_dashboard",
		New: func() caddy.Module { return new(Dashboard) },
	}
}

// Provision sets up the dashboard.
func (d *Dashboard) Provision(ctx caddy.Context) error {
	if d.Path == "" {
		d.Path = "/dashboard"
	}
	if d.UpdateInterval == 0 {
		d.UpdateInterval = caddy.Duration(time.Second)
	}
	return nil
}

// ServeHTTP handles HTTP requests for the dashboard.
func (d *Dashboard) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if r.URL.Path == d.Path {
		d.serveDashboard(w, r)
		return nil
	}
	if r.URL.Path == d.Path+"/events" {
		d.handleSSE(w, r)
		return nil
	}
	return next.ServeHTTP(w, r)
}

// serveDashboard serves the main dashboard HTML.
func (d *Dashboard) serveDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(staticFS, "static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, nil)
}

// handleSSE handles Server-Sent Events for real-time updates.
func (d *Dashboard) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(time.Duration(d.UpdateInterval))
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			data := d.collectTrafficData()
			jsonData, err := json.Marshal(data)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		}
	}
}

// collectTrafficData gathers traffic metrics.
func (d *Dashboard) collectTrafficData() TrafficData {
	// In a real implementation, collect from Caddy's metrics
	// For now, use stub data
	return TrafficData{
		Timestamp:   time.Now(),
		RequestRate: 100.5, // requests per second
		ErrorRate:   2.3,   // errors per second
		Latency: map[string]float64{
			"p50": 50.0,
			"p95": 200.0,
			"p99": 500.0,
		},
		ActiveConns: 42,
		GeoData: map[string]int{
			"US": 25,
			"EU": 10,
			"AS": 7,
		},
	}
}

// UnmarshalCaddyfile sets up the module from Caddyfile tokens.
func (d *Dashboard) UnmarshalCaddyfile(c *caddyfile.Dispenser) error {
	c.Next() // consume handler name
	if c.NextArg() {
		return c.ArgErr()
	}
	for c.NextBlock(0) {
		switch c.Val() {
		case "path":
			if !c.NextArg() {
				return c.ArgErr()
			}
			d.Path = c.Val()
		case "update_interval":
			if !c.NextArg() {
				return c.ArgErr()
			}
			dur, err := caddy.ParseDuration(c.Val())
			if err != nil {
				return c.Errf("invalid update_interval: %v", err)
			}
			d.UpdateInterval = caddy.Duration(dur)
		default:
			return c.Errf("unrecognized option '%s'", c.Val())
		}
	}
	return nil
}

// parseDashboard parses the traffic_dashboard directive.
func parseDashboard(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	d := new(Dashboard)
	err := d.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*Dashboard)(nil)
	_ caddy.Provisioner           = (*Dashboard)(nil)
	_ caddyfile.Unmarshaler       = (*Dashboard)(nil)
)