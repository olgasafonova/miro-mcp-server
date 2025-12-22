package miro

import (
	"errors"
	"sync"
	"time"
)

// =============================================================================
// Circuit Breaker Configuration
// =============================================================================

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit.
	FailureThreshold int

	// SuccessThreshold is the number of successes needed to close the circuit.
	SuccessThreshold int

	// Timeout is how long to wait before allowing a test request.
	Timeout time.Duration

	// MaxHalfOpenRequests is the number of test requests allowed in half-open state.
	MaxHalfOpenRequests int
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 1,
	}
}

// =============================================================================
// Circuit Breaker States
// =============================================================================

// CircuitState represents the current state of the circuit breaker.
type CircuitState int

const (
	// CircuitClosed means requests are allowed.
	CircuitClosed CircuitState = iota
	// CircuitOpen means requests are blocked.
	CircuitOpen
	// CircuitHalfOpen means a test request is allowed.
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// =============================================================================
// Circuit Breaker Errors
// =============================================================================

// ErrCircuitOpen is returned when the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// ErrTooManyRequests is returned when too many half-open requests are in progress.
var ErrTooManyRequests = errors.New("too many requests in half-open state")

// =============================================================================
// Circuit Breaker
// =============================================================================

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu     sync.RWMutex
	config CircuitBreakerConfig

	state            CircuitState
	failures         int
	successes        int
	lastFailureTime  time.Time
	halfOpenRequests int

	// Stats
	totalRequests   int64
	totalFailures   int64
	totalSuccesses  int64
	circuitOpenings int64
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.getState()
}

// getState returns the current state, checking for timeout transitions (must hold lock).
func (cb *CircuitBreaker) getState() CircuitState {
	if cb.state == CircuitOpen {
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			return CircuitHalfOpen
		}
	}
	return cb.state
}

// Allow checks if a request should be allowed.
// Returns an error if the circuit is open or too many half-open requests.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalRequests++

	state := cb.getState()

	switch state {
	case CircuitClosed:
		return nil

	case CircuitOpen:
		return ErrCircuitOpen

	case CircuitHalfOpen:
		if cb.halfOpenRequests >= cb.config.MaxHalfOpenRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenRequests++
		return nil
	}

	return nil
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalSuccesses++
	state := cb.getState()

	switch state {
	case CircuitClosed:
		cb.failures = 0 // Reset failure count on success

	case CircuitHalfOpen:
		cb.halfOpenRequests--
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			// Close the circuit
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalFailures++
	cb.lastFailureTime = time.Now()
	state := cb.getState()

	switch state {
	case CircuitClosed:
		cb.failures++
		if cb.failures >= cb.config.FailureThreshold {
			// Open the circuit
			cb.state = CircuitOpen
			cb.circuitOpenings++
		}

	case CircuitHalfOpen:
		// Immediately open the circuit again on failure
		cb.halfOpenRequests--
		cb.state = CircuitOpen
		cb.successes = 0
		cb.circuitOpenings++
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
}

// CircuitBreakerStats holds statistics for the circuit breaker.
type CircuitBreakerStats struct {
	State           string
	TotalRequests   int64
	TotalSuccesses  int64
	TotalFailures   int64
	CircuitOpenings int64
	CurrentFailures int
}

// Stats returns statistics for the circuit breaker.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:           cb.getState().String(),
		TotalRequests:   cb.totalRequests,
		TotalSuccesses:  cb.totalSuccesses,
		TotalFailures:   cb.totalFailures,
		CircuitOpenings: cb.circuitOpenings,
		CurrentFailures: cb.failures,
	}
}

// =============================================================================
// Circuit Breaker Registry
// =============================================================================

// CircuitBreakerRegistry manages circuit breakers for different endpoints.
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerRegistry creates a new registry with the given configuration.
func NewCircuitBreakerRegistry(config CircuitBreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Get returns the circuit breaker for the given endpoint, creating one if needed.
func (r *CircuitBreakerRegistry) Get(endpoint string) *CircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[endpoint]
	r.mu.RUnlock()

	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok = r.breakers[endpoint]; ok {
		return cb
	}

	cb = NewCircuitBreaker(r.config)
	r.breakers[endpoint] = cb
	return cb
}

// AllStats returns statistics for all circuit breakers.
func (r *CircuitBreakerRegistry) AllStats() map[string]CircuitBreakerStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats, len(r.breakers))
	for endpoint, cb := range r.breakers {
		stats[endpoint] = cb.Stats()
	}
	return stats
}

// Reset resets all circuit breakers.
func (r *CircuitBreakerRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cb := range r.breakers {
		cb.Reset()
	}
}
