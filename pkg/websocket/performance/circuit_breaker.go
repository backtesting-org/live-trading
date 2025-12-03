package performance

import (
	"fmt"
	"sync"
	"time"
)

type circuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailure  time.Time
	state        string // "closed", "open", "half-open"
	mutex        sync.Mutex
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) CircuitBreaker {
	return &circuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

// Execute executes the given function through the circuit breaker (alias for Call)
func (cb *circuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
			cb.failures = 0
		} else {
			return fmt.Errorf("circuit breaker open")
		}
	}

	err := fn()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.maxFailures {
			cb.state = "open"
		}
		return err
	}

	if cb.state == "half-open" {
		cb.state = "closed"
	}
	cb.failures = 0
	return nil
}

func (cb *circuitBreaker) GetState() string {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}
