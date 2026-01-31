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

package circuitbreakers

import (
	"math"
	"sync"
	"time"

	"github.com/Pritam499/proxflow/v2"
)

func init() {
	caddy.RegisterModule(IntelligentCircuitBreaker{})
}

// IntelligentCircuitBreaker implements an intelligent circuit breaker
// that monitors multiple metrics and provides auto-recovery with traffic ramp-up.
type IntelligentCircuitBreaker struct {
	// FailureThreshold is the failure rate threshold (0-1) to open the circuit.
	FailureThreshold float64 `json:"failure_threshold,omitempty"`

	// RecoveryTimeout is how long to wait before attempting recovery.
	RecoveryTimeout caddy.Duration `json:"recovery_timeout,omitempty"`

	// HalfOpenMaxRequests is the maximum requests allowed in half-open state.
	HalfOpenMaxRequests int `json:"half_open_max_requests,omitempty"`

	// LatencyThreshold is the latency threshold to consider a request slow.
	LatencyThreshold caddy.Duration `json:"latency_threshold,omitempty"`

	// MinRequests is the minimum requests needed to evaluate metrics.
	MinRequests int `json:"min_requests,omitempty"`

	// EvaluationWindow is the time window for metric evaluation.
	EvaluationWindow caddy.Duration `json:"evaluation_window,omitempty"`

	// RampUpDuration is how long the ramp-up phase lasts.
	RampUpDuration caddy.Duration `json:"ramp_up_duration,omitempty"`

	state            State
	stateMu          sync.RWMutex
	requests         []RequestRecord
	requestsMu       sync.Mutex
	lastFailureTime  time.Time
	halfOpenRequests int
	rampUpStart      time.Time
}

// State represents the circuit breaker state.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
	StateRampUp
)

// RequestRecord holds information about a request.
type RequestRecord struct {
	Timestamp time.Time
	Success   bool
	Latency   time.Duration
}

// CaddyModule returns the Caddy module information.
func (IntelligentCircuitBreaker) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.reverse_proxy.circuit_breakers.intelligent",
		New: func() caddy.Module { return new(IntelligentCircuitBreaker) },
	}
}

// Provision sets up the circuit breaker.
func (cb *IntelligentCircuitBreaker) Provision(ctx caddy.Context) error {
	if cb.FailureThreshold == 0 {
		cb.FailureThreshold = 0.5
	}
	if cb.RecoveryTimeout == 0 {
		cb.RecoveryTimeout = caddy.Duration(60 * time.Second)
	}
	if cb.HalfOpenMaxRequests == 0 {
		cb.HalfOpenMaxRequests = 3
	}
	if cb.LatencyThreshold == 0 {
		cb.LatencyThreshold = caddy.Duration(5 * time.Second)
	}
	if cb.MinRequests == 0 {
		cb.MinRequests = 10
	}
	if cb.EvaluationWindow == 0 {
		cb.EvaluationWindow = caddy.Duration(60 * time.Second)
	}
	if cb.RampUpDuration == 0 {
		cb.RampUpDuration = caddy.Duration(30 * time.Second)
	}
	cb.state = StateClosed
	return nil
}

// OK returns whether the circuit breaker allows requests.
func (cb *IntelligentCircuitBreaker) OK() bool {
	cb.stateMu.RLock()
	defer cb.stateMu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > time.Duration(cb.RecoveryTimeout) {
			cb.stateMu.RUnlock()
			cb.stateMu.Lock()
			cb.state = StateHalfOpen
			cb.halfOpenRequests = 0
			cb.stateMu.Unlock()
			cb.stateMu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return cb.halfOpenRequests < cb.HalfOpenMaxRequests
	case StateRampUp:
		return true
	default:
		return false
	}
}

// RecordMetric records a request metric.
func (cb *IntelligentCircuitBreaker) RecordMetric(statusCode int, latency time.Duration) {
	isSuccess := statusCode >= 200 && statusCode < 500 && latency < time.Duration(cb.LatencyThreshold)
	isTimeout := latency >= time.Duration(cb.LatencyThreshold)

	cb.requestsMu.Lock()
	cb.requests = append(cb.requests, RequestRecord{
		Timestamp: time.Now(),
		Success:   isSuccess,
		Latency:   latency,
	})
	// Keep only recent requests
	cutoff := time.Now().Add(-time.Duration(cb.EvaluationWindow))
	for i, req := range cb.requests {
		if req.Timestamp.After(cutoff) {
			cb.requests = cb.requests[i:]
			break
		}
	}
	cb.requestsMu.Unlock()

	cb.stateMu.Lock()
	defer cb.stateMu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.halfOpenRequests++
		if isSuccess {
			if cb.halfOpenRequests >= cb.HalfOpenMaxRequests {
				cb.state = StateRampUp
				cb.rampUpStart = time.Now()
			}
		} else {
			cb.state = StateOpen
			cb.lastFailureTime = time.Now()
		}
	case StateRampUp:
		if !isSuccess || isTimeout {
			cb.state = StateOpen
			cb.lastFailureTime = time.Now()
		} else if time.Since(cb.rampUpStart) >= time.Duration(cb.RampUpDuration) {
			cb.state = StateClosed
		}
	case StateClosed:
		if len(cb.requests) >= cb.MinRequests {
			failureRate := cb.calculateFailureRate()
			if failureRate > cb.FailureThreshold {
				cb.state = StateOpen
				cb.lastFailureTime = time.Now()
			}
		}
	}
}

// calculateFailureRate calculates the failure rate from recent requests.
func (cb *IntelligentCircuitBreaker) calculateFailureRate() float64 {
	cb.requestsMu.Lock()
	defer cb.requestsMu.Unlock()

	if len(cb.requests) == 0 {
		return 0
	}

	failures := 0
	for _, req := range cb.requests {
		if !req.Success {
			failures++
		}
	}

	return float64(failures) / float64(len(cb.requests))
}

// RampUpFactor returns the traffic ramp-up factor (0-1) during ramp-up phase.
func (cb *IntelligentCircuitBreaker) RampUpFactor() float64 {
	cb.stateMu.RLock()
	defer cb.stateMu.RUnlock()

	if cb.state != StateRampUp {
		return 1.0
	}

	elapsed := time.Since(cb.rampUpStart)
	total := time.Duration(cb.RampUpDuration)
	if elapsed >= total {
		return 1.0
	}

	// Exponential ramp-up
	progress := float64(elapsed) / float64(total)
	return 1 - math.Exp(-3*progress) // Smooth curve from 0 to 1
}

// StateString returns the current state as a string.
func (cb *IntelligentCircuitBreaker) StateString() string {
	cb.stateMu.RLock()
	defer cb.stateMu.RUnlock()

	switch cb.state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	case StateRampUp:
		return "ramp-up"
	default:
		return "unknown"
	}
}