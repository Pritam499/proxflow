package autoscaling

import (
	"net/http"
	"testing"
	"time"

	"github.com/Pritam499/proxflow/v2"
)

func TestAutoScalingUpstreams_Provision(t *testing.T) {
	a := &AutoScalingUpstreams{
		ServiceDiscovery: &ServiceDiscovery{
			Type: "static",
			Static: &StaticConfig{
				Addresses: []string{"localhost:8080"},
			},
		},
	}

	ctx := caddy.ActiveContext()
	err := a.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if a.HealthChecks == nil {
		t.Error("HealthChecks should be initialized")
	}

	if a.Scaling == nil {
		t.Error("Scaling should be initialized")
	}
}

func TestAutoScalingUpstreams_GetUpstreams(t *testing.T) {
	a := &AutoScalingUpstreams{
		ServiceDiscovery: &ServiceDiscovery{
			Type: "static",
			Static: &StaticConfig{
				Addresses: []string{"localhost:8080", "localhost:8081"},
			},
		},
		Backends: []Backend{
			{Address: "localhost:8080", Healthy: true},
			{Address: "localhost:8081", Healthy: false},
		},
	}

	req, _ := http.NewRequest("GET", "/test", nil)
	upstreams, err := a.GetUpstreams(req)
	if err != nil {
		t.Fatalf("GetUpstreams failed: %v", err)
	}

	if len(upstreams) != 1 {
		t.Errorf("Expected 1 healthy upstream, got %d", len(upstreams))
	}

	if upstreams[0].Dial != "localhost:8080" {
		t.Errorf("Expected upstream address localhost:8080, got %s", upstreams[0].Dial)
	}
}

func TestAutoScalingUpstreams_shouldScaleUp(t *testing.T) {
	a := &AutoScalingUpstreams{
		Scaling: &ScalingConfig{
			MinBackends:      1,
			MaxBackends:      5,
			ScaleUpThreshold: 0.8,
		},
		Backends: []Backend{
			{Address: "localhost:8080", Healthy: true, Load: 0.9}, // High load
		},
	}

	if !a.shouldScaleUp() {
		t.Error("Should scale up due to high load")
	}

	// Test with low load
	a.Backends[0].Load = 0.1
	if a.shouldScaleUp() {
		t.Error("Should not scale up with low load")
	}

	// Test at max backends
	a.Scaling.MaxBackends = 1
	a.Backends[0].Load = 0.9
	if a.shouldScaleUp() {
		t.Error("Should not scale up at max backends")
	}
}

func TestAutoScalingUpstreams_shouldScaleDown(t *testing.T) {
	a := &AutoScalingUpstreams{
		Scaling: &ScalingConfig{
			MinBackends:        1,
			MaxBackends:        5,
			ScaleDownThreshold: 0.2,
		},
		Backends: []Backend{
			{Address: "localhost:8080", Healthy: true, Load: 0.1}, // Low load
			{Address: "localhost:8081", Healthy: true, Load: 0.1},
		},
	}

	if !a.shouldScaleDown() {
		t.Error("Should scale down due to low load")
	}

	// Test with high load
	a.Backends[0].Load = 0.8
	a.Backends[1].Load = 0.8
	if a.shouldScaleDown() {
		t.Error("Should not scale down with high load")
	}

	// Test at min backends
	a.Scaling.MinBackends = 2
	a.Backends[0].Load = 0.1
	a.Backends[1].Load = 0.1
	if a.shouldScaleDown() {
		t.Error("Should not scale down at min backends")
	}
}