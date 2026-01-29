package circuitbreakers

import (
	"testing"
	"time"

	"github.com/Pritam499/proxflow/v2"
)

func TestIntelligentCircuitBreaker_Provision(t *testing.T) {
	cb := &IntelligentCircuitBreaker{}
	ctx := caddy.ActiveContext()
	err := cb.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if cb.FailureThreshold != 0.5 {
		t.Errorf("Expected FailureThreshold 0.5, got %f", cb.FailureThreshold)
	}
	if cb.state != StateClosed {
		t.Errorf("Expected state Closed, got %d", cb.state)
	}
}

func TestIntelligentCircuitBreaker_OK(t *testing.T) {
	cb := &IntelligentCircuitBreaker{}
	cb.Provision(caddy.ActiveContext())

	// Initially closed
	if !cb.OK() {
		t.Error("Expected OK in closed state")
	}

	// Simulate failures to open
	for i := 0; i < 15; i++ {
		cb.RecordMetric(500, 100*time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond) // Allow evaluation

	if cb.OK() {
		t.Error("Expected not OK in open state")
	}

	// Wait for recovery timeout
	cb.RecoveryTimeout = caddy.Duration(10 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	if !cb.OK() {
		t.Error("Expected OK after recovery timeout")
	}
}

func TestIntelligentCircuitBreaker_RampUp(t *testing.T) {
	cb := &IntelligentCircuitBreaker{}
	cb.Provision(caddy.ActiveContext())

	// Force ramp-up state
	cb.state = StateRampUp
	cb.rampUpStart = time.Now()
	cb.RampUpDuration = caddy.Duration(100 * time.Millisecond)

	factor := cb.RampUpFactor()
	if factor <= 0 || factor > 1 {
		t.Errorf("RampUpFactor should be between 0 and 1, got %f", factor)
	}

	time.Sleep(150 * time.Millisecond)
	factor = cb.RampUpFactor()
	if factor != 1.0 {
		t.Errorf("RampUpFactor should be 1.0 after duration, got %f", factor)
	}
}