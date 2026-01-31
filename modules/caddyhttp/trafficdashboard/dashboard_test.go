package trafficdashboard

import (
	"testing"

	"github.com/Pritam499/proxflow/v2"
)

func TestDashboard_Provision(t *testing.T) {
	d := &Dashboard{}
	ctx := caddy.ActiveContext()
	err := d.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if d.Path != "/dashboard" {
		t.Errorf("Expected path '/dashboard', got %s", d.Path)
	}
}

func TestDashboard_collectTrafficData(t *testing.T) {
	d := &Dashboard{}
	data := d.collectTrafficData()

	if data.RequestRate <= 0 {
		t.Error("RequestRate should be positive")
	}
	if data.ErrorRate < 0 {
		t.Error("ErrorRate should be non-negative")
	}
	if data.ActiveConns < 0 {
		t.Error("ActiveConns should be non-negative")
	}
	if data.Latency == nil {
		t.Error("Latency should not be nil")
	}
	if data.GeoData == nil {
		t.Error("GeoData should not be nil")
	}
}