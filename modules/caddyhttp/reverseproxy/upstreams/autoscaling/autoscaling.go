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

package autoscaling

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Pritam499/proxflow/v2"
	"github.com/Pritam499/proxflow/v2/caddyconfig/caddyfile"
	"github.com/Pritam499/proxflow/v2/modules/caddyhttp/reverseproxy"
)

func init() {
	caddy.RegisterModule(AutoScalingUpstreams{})
}

// AutoScalingUpstreams provides upstreams with automatic scaling and health monitoring.
type AutoScalingUpstreams struct {
	// ServiceDiscovery specifies the service discovery mechanism.
	ServiceDiscovery *ServiceDiscovery `json:"service_discovery,omitempty"`

	// HealthChecks configures health checking for backends.
	HealthChecks *HealthCheckConfig `json:"health_checks,omitempty"`

	// Scaling configures auto-scaling behavior.
	Scaling *ScalingConfig `json:"scaling,omitempty"`

	// Backends is the current list of backends (managed internally).
	Backends []Backend `json:"-"`

	mu       sync.RWMutex
	cancel   context.CancelFunc
	done     chan struct{}
}

// ServiceDiscovery configures service discovery.
type ServiceDiscovery struct {
	// Type can be "kubernetes", "docker_swarm", "consul", "static".
	Type string `json:"type,omitempty"`

	// Kubernetes config.
	Kubernetes *KubernetesConfig `json:"kubernetes,omitempty"`

	// Docker Swarm config.
	DockerSwarm *DockerSwarmConfig `json:"docker_swarm,omitempty"`

	// Consul config.
	Consul *ConsulConfig `json:"consul,omitempty"`

	// Static config for manual backend management.
	Static *StaticConfig `json:"static,omitempty"`
}

// KubernetesConfig for Kubernetes service discovery.
type KubernetesConfig struct {
	// Namespace to watch.
	Namespace string `json:"namespace,omitempty"`

	// ServiceName to discover.
	ServiceName string `json:"service_name,omitempty"`

	// LabelSelector for pod selection.
	LabelSelector string `json:"label_selector,omitempty"`
}

// DockerSwarmConfig for Docker Swarm service discovery.
type DockerSwarmConfig struct {
	// ServiceName to discover.
	ServiceName string `json:"service_name,omitempty"`

	// NetworkName for service discovery.
	NetworkName string `json:"network_name,omitempty"`
}

// ConsulConfig for Consul service discovery.
type ConsulConfig struct {
	// ServiceName to discover.
	ServiceName string `json:"service_name,omitempty"`

	// Address of Consul agent.
	Address string `json:"address,omitempty"`

	// Token for Consul authentication.
	Token string `json:"token,omitempty"`
}

// StaticConfig for static backend management.
type StaticConfig struct {
	// Addresses is the list of backend addresses.
	Addresses []string `json:"addresses,omitempty"`
}

// HealthCheckConfig configures health checking.
type HealthCheckConfig struct {
	// Path for health check endpoint.
	Path string `json:"path,omitempty"`

	// Interval between health checks.
	Interval caddy.Duration `json:"interval,omitempty"`

	// Timeout for health check requests.
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// UnhealthyThreshold number of failures to mark unhealthy.
	UnhealthyThreshold int `json:"unhealthy_threshold,omitempty"`

	// HealthyThreshold number of successes to mark healthy.
	HealthyThreshold int `json:"healthy_threshold,omitempty"`

	// PerBackend configs override global health checks.
	PerBackend map[string]*HealthCheckConfig `json:"per_backend,omitempty"`
}

// ScalingConfig configures auto-scaling behavior.
type ScalingConfig struct {
	// MinBackends minimum number of backends to maintain.
	MinBackends int `json:"min_backends,omitempty"`

	// MaxBackends maximum number of backends to allow.
	MaxBackends int `json:"max_backends,omitempty"`

	// ScaleUpThreshold CPU/memory usage threshold to trigger scale up.
	ScaleUpThreshold float64 `json:"scale_up_threshold,omitempty"`

	// ScaleDownThreshold usage threshold to trigger scale down.
	ScaleDownThreshold float64 `json:"scale_down_threshold,omitempty"`

	// ScaleCooldown time to wait between scaling operations.
	ScaleCooldown caddy.Duration `json:"scale_cooldown,omitempty"`

	// Provider for scaling operations (kubernetes, docker_swarm, etc.).
	Provider string `json:"provider,omitempty"`
}

// Backend represents a backend instance.
type Backend struct {
	// Address is the backend address.
	Address string `json:"address"`

	// Healthy indicates if the backend is healthy.
	Healthy bool `json:"healthy"`

	// Load represents current load (0-1).
	Load float64 `json:"load"`

	// LastHealthCheck time of last health check.
	LastHealthCheck time.Time `json:"last_health_check"`

	// FailureCount number of consecutive failures.
	FailureCount int `json:"failure_count"`

	// SuccessCount number of consecutive successes.
	SuccessCount int `json:"success_count"`
}

// CaddyModule returns the Caddy module information.
func (AutoScalingUpstreams) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.reverse_proxy.upstreams.auto_scaling",
		New: func() caddy.Module { return new(AutoScalingUpstreams) },
	}
}

// Provision sets up the auto-scaling upstreams.
func (a *AutoScalingUpstreams) Provision(ctx caddy.Context) error {
	if a.ServiceDiscovery == nil {
		return fmt.Errorf("service_discovery is required")
	}

	if a.HealthChecks == nil {
		a.HealthChecks = &HealthCheckConfig{
			Path:               "/health",
			Interval:           caddy.Duration(30 * time.Second),
			Timeout:            caddy.Duration(5 * time.Second),
			UnhealthyThreshold: 3,
			HealthyThreshold:   2,
		}
	}

	if a.Scaling == nil {
		a.Scaling = &ScalingConfig{
			MinBackends:       1,
			MaxBackends:       10,
			ScaleUpThreshold:  0.8,
			ScaleDownThreshold: 0.2,
			ScaleCooldown:     caddy.Duration(5 * time.Minute),
		}
	}

	a.done = make(chan struct{})

	// Start background processes
	go a.discoveryLoop()
	go a.healthCheckLoop()
	go a.scalingLoop()

	return nil
}

// GetUpstreams returns the current healthy upstreams.
func (a *AutoScalingUpstreams) GetUpstreams(r *http.Request) ([]*reverseproxy.Upstream, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var upstreams []*reverseproxy.Upstream
	for _, backend := range a.Backends {
		if backend.Healthy {
			upstream := &reverseproxy.Upstream{
				Dial: backend.Address,
			}
			upstreams = append(upstreams, upstream)
		}
	}

	return upstreams, nil
}

// discoveryLoop continuously discovers backends.
func (a *AutoScalingUpstreams) discoveryLoop() {
	ticker := time.NewTicker(30 * time.Second) // Discovery interval
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			a.discoverBackends()
		}
	}
}

// discoverBackends discovers backends using configured service discovery.
func (a *AutoScalingUpstreams) discoverBackends() {
	var addresses []string
	var err error

	switch a.ServiceDiscovery.Type {
	case "kubernetes":
		addresses, err = a.discoverKubernetes()
	case "docker_swarm":
		addresses, err = a.discoverDockerSwarm()
	case "consul":
		addresses, err = a.discoverConsul()
	case "static":
		addresses = a.ServiceDiscovery.Static.Addresses
	default:
		// Fallback to static
		addresses = a.ServiceDiscovery.Static.Addresses
	}

	if err != nil {
		// Log error
		return
	}

	a.updateBackends(addresses)
}

// updateBackends updates the backend list.
func (a *AutoScalingUpstreams) updateBackends(addresses []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create new backends map
	newBackends := make(map[string]*Backend)

	// Add existing backends
	for i := range a.Backends {
		backend := &a.Backends[i]
		newBackends[backend.Address] = backend
	}

	// Add new backends
	for _, addr := range addresses {
		if _, exists := newBackends[addr]; !exists {
			newBackends[addr] = &Backend{
				Address:  addr,
				Healthy:  true, // Assume healthy initially
				Load:     0.0,
			}
		}
	}

	// Convert map to slice
	a.Backends = make([]Backend, 0, len(newBackends))
	for _, backend := range newBackends {
		a.Backends = append(a.Backends, *backend)
	}
}

// healthCheckLoop performs continuous health checks.
func (a *AutoScalingUpstreams) healthCheckLoop() {
	ticker := time.NewTicker(time.Duration(a.HealthChecks.Interval))
	defer ticker.Stop()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			a.performHealthChecks()
		}
	}
}

// performHealthChecks checks health of all backends.
func (a *AutoScalingUpstreams) performHealthChecks() {
	a.mu.Lock()
	defer a.mu.Unlock()

	client := &http.Client{
		Timeout: time.Duration(a.HealthChecks.Timeout),
	}

	for i := range a.Backends {
		backend := &a.Backends[i]
		a.checkBackendHealth(client, backend)
	}
}

// checkBackendHealth checks health of a single backend.
func (a *AutoScalingUpstreams) checkBackendHealth(client *http.Client, backend *Backend) {
	healthURL := fmt.Sprintf("http://%s%s", backend.Address, a.HealthChecks.Path)

	resp, err := client.Get(healthURL)
	backend.LastHealthCheck = time.Now()

	if err != nil {
		backend.FailureCount++
		backend.SuccessCount = 0
		if backend.FailureCount >= a.HealthChecks.UnhealthyThreshold {
			backend.Healthy = false
		}
		return
	}

	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		backend.SuccessCount++
		backend.FailureCount = 0
		if backend.SuccessCount >= a.HealthChecks.HealthyThreshold {
			backend.Healthy = true
		}
	} else {
		backend.FailureCount++
		backend.SuccessCount = 0
		if backend.FailureCount >= a.HealthChecks.UnhealthyThreshold {
			backend.Healthy = false
		}
	}
}

// scalingLoop performs auto-scaling operations.
func (a *AutoScalingUpstreams) scalingLoop() {
	ticker := time.NewTicker(1 * time.Minute) // Scaling check interval
	defer ticker.Stop()

	lastScaleTime := time.Now()

	for {
		select {
		case <-a.done:
			return
		case <-ticker.C:
			if time.Since(lastScaleTime) > time.Duration(a.Scaling.ScaleCooldown) {
				if a.shouldScaleUp() {
					a.scaleUp()
					lastScaleTime = time.Now()
				} else if a.shouldScaleDown() {
					a.scaleDown()
					lastScaleTime = time.Now()
				}
			}
		}
	}
}

// shouldScaleUp determines if scaling up is needed.
func (a *AutoScalingUpstreams) shouldScaleUp() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.Backends) >= a.Scaling.MaxBackends {
		return false
	}

	healthyCount := 0
	totalLoad := 0.0

	for _, backend := range a.Backends {
		if backend.Healthy {
			healthyCount++
			totalLoad += backend.Load
		}
	}

	if healthyCount == 0 {
		return true // Scale up if no healthy backends
	}

	avgLoad := totalLoad / float64(healthyCount)
	return avgLoad > a.Scaling.ScaleUpThreshold
}

// shouldScaleDown determines if scaling down is needed.
func (a *AutoScalingUpstreams) shouldScaleDown() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.Backends) <= a.Scaling.MinBackends {
		return false
	}

	totalLoad := 0.0
	healthyCount := 0

	for _, backend := range a.Backends {
		if backend.Healthy {
			healthyCount++
			totalLoad += backend.Load
		}
	}

	if healthyCount == 0 {
		return false
	}

	avgLoad := totalLoad / float64(healthyCount)
	return avgLoad < a.Scaling.ScaleDownThreshold
}

// scaleUp adds more backends.
func (a *AutoScalingUpstreams) scaleUp() {
	// Implementation depends on the scaling provider
	switch a.Scaling.Provider {
	case "kubernetes":
		a.scaleUpKubernetes()
	case "docker_swarm":
		a.scaleUpDockerSwarm()
	default:
		// For demo, just mark that we would scale up
	}
}

// scaleDown removes backends.
func (a *AutoScalingUpstreams) scaleDown() {
	// Implementation depends on the scaling provider
	switch a.Scaling.Provider {
	case "kubernetes":
		a.scaleDownKubernetes()
	case "docker_swarm":
		a.scaleDownDockerSwarm()
	default:
		// For demo, just mark that we would scale down
	}
}

// Placeholder methods for service discovery and scaling
func (a *AutoScalingUpstreams) discoverKubernetes() ([]string, error) {
	// Kubernetes service discovery implementation
	return []string{"backend-1:8080", "backend-2:8080"}, nil
}

func (a *AutoScalingUpstreams) discoverDockerSwarm() ([]string, error) {
	// Docker Swarm service discovery implementation
	return []string{"backend-1:8080", "backend-2:8080"}, nil
}

func (a *AutoScalingUpstreams) discoverConsul() ([]string, error) {
	// Consul service discovery implementation
	return []string{"backend-1:8080", "backend-2:8080"}, nil
}

func (a *AutoScalingUpstreams) scaleUpKubernetes() {
	// Kubernetes scaling implementation
}

func (a *AutoScalingUpstreams) scaleDownKubernetes() {
	// Kubernetes scaling implementation
}

func (a *AutoScalingUpstreams) scaleUpDockerSwarm() {
	// Docker Swarm scaling implementation
}

func (a *AutoScalingUpstreams) scaleDownDockerSwarm() {
	// Docker Swarm scaling implementation
}

// Cleanup stops background processes.
func (a *AutoScalingUpstreams) Cleanup() error {
	if a.done != nil {
		close(a.done)
	}
	return nil
}

// UnmarshalCaddyfile parses the auto_scaling directive.
func (a *AutoScalingUpstreams) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next() // consume directive name

	for d.NextBlock(0) {
		switch d.Val() {
		case "service_discovery":
			sd := &ServiceDiscovery{}
			for d.NextBlock(1) {
				switch d.Val() {
				case "type":
					if !d.NextArg() {
						return d.ArgErr()
					}
					sd.Type = d.Val()
				case "kubernetes":
					k8s := &KubernetesConfig{}
					for d.NextBlock(2) {
						switch d.Val() {
						case "namespace":
							if !d.NextArg() {
								return d.ArgErr()
							}
							k8s.Namespace = d.Val()
						case "service_name":
							if !d.NextArg() {
								return d.ArgErr()
							}
							k8s.ServiceName = d.Val()
						case "label_selector":
							if !d.NextArg() {
								return d.ArgErr()
							}
							k8s.LabelSelector = d.Val()
						}
					}
					sd.Kubernetes = k8s
				case "consul":
					consul := &ConsulConfig{}
					for d.NextBlock(2) {
						switch d.Val() {
						case "service_name":
							if !d.NextArg() {
								return d.ArgErr()
							}
							consul.ServiceName = d.Val()
						case "address":
							if !d.NextArg() {
								return d.ArgErr()
							}
							consul.Address = d.Val()
						case "token":
							if !d.NextArg() {
								return d.ArgErr()
							}
							consul.Token = d.Val()
						}
					}
					sd.Consul = consul
				case "static":
					static := &StaticConfig{}
					for d.NextBlock(2) {
						switch d.Val() {
						case "address":
							args := d.RemainingArgs()
							static.Addresses = append(static.Addresses, args...)
						}
					}
					sd.Static = static
				}
			}
			a.ServiceDiscovery = sd
		case "health_checks":
			hc := &HealthCheckConfig{}
			for d.NextBlock(1) {
				switch d.Val() {
				case "path":
					if !d.NextArg() {
						return d.ArgErr()
					}
					hc.Path = d.Val()
				case "interval":
					if !d.NextArg() {
						return d.ArgErr()
					}
					interval, err := caddy.ParseDuration(d.Val())
					if err != nil {
						return d.Errf("invalid interval: %v", err)
					}
					hc.Interval = caddy.Duration(interval)
				case "timeout":
					if !d.NextArg() {
						return d.ArgErr()
					}
					timeout, err := caddy.ParseDuration(d.Val())
					if err != nil {
						return d.Errf("invalid timeout: %v", err)
					}
					hc.Timeout = caddy.Duration(timeout)
				case "unhealthy_threshold":
					if !d.NextArg() {
						return d.ArgErr()
					}
					threshold, err := strconv.Atoi(d.Val())
					if err != nil {
						return d.Errf("invalid unhealthy_threshold: %v", err)
					}
					hc.UnhealthyThreshold = threshold
				case "healthy_threshold":
					if !d.NextArg() {
						return d.ArgErr()
					}
					threshold, err := strconv.Atoi(d.Val())
					if err != nil {
						return d.Errf("invalid healthy_threshold: %v", err)
					}
					hc.HealthyThreshold = threshold
				}
			}
			a.HealthChecks = hc
		case "scaling":
			sc := &ScalingConfig{}
			for d.NextBlock(1) {
				switch d.Val() {
				case "min_backends":
					if !d.NextArg() {
						return d.ArgErr()
					}
					min, err := strconv.Atoi(d.Val())
					if err != nil {
						return d.Errf("invalid min_backends: %v", err)
					}
					sc.MinBackends = min
				case "max_backends":
					if !d.NextArg() {
						return d.ArgErr()
					}
					max, err := strconv.Atoi(d.Val())
					if err != nil {
						return d.Errf("invalid max_backends: %v", err)
					}
					sc.MaxBackends = max
				case "scale_up_threshold":
					if !d.NextArg() {
						return d.ArgErr()
					}
					threshold, err := strconv.ParseFloat(d.Val(), 64)
					if err != nil {
						return d.Errf("invalid scale_up_threshold: %v", err)
					}
					sc.ScaleUpThreshold = threshold
				case "scale_down_threshold":
					if !d.NextArg() {
						return d.ArgErr()
					}
					threshold, err := strconv.ParseFloat(d.Val(), 64)
					if err != nil {
						return d.Errf("invalid scale_down_threshold: %v", err)
					}
					sc.ScaleDownThreshold = threshold
				case "scale_cooldown":
					if !d.NextArg() {
						return d.ArgErr()
					}
					cooldown, err := caddy.ParseDuration(d.Val())
					if err != nil {
						return d.Errf("invalid scale_cooldown: %v", err)
					}
					sc.ScaleCooldown = caddy.Duration(cooldown)
				case "provider":
					if !d.NextArg() {
						return d.ArgErr()
					}
					sc.Provider = d.Val()
				}
			}
			a.Scaling = sc
		default:
			return d.Errf("unrecognized option '%s'", d.Val())
		}
	}
	return nil
}

// Interface guards
var (
	_ reverseproxy.UpstreamSource = (*AutoScalingUpstreams)(nil)
	_ caddy.Provisioner           = (*AutoScalingUpstreams)(nil)
	_ caddy.CleanerUpper          = (*AutoScalingUpstreams)(nil)
	_ caddyfile.Unmarshaler       = (*AutoScalingUpstreams)(nil)
)