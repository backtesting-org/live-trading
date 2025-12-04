package performance

import "time"

// Metrics defines metrics collection operations
type Metrics interface {
	IncrementReceived()
	IncrementProcessed(latency time.Duration)
	IncrementDropped()
	IncrementConnectionError()
	IncrementReconnection()
	GetStats() map[string]interface{}
}

// CircuitBreaker defines circuit breaker operations
type CircuitBreaker interface {
	Execute(fn func() error) error
	GetState() string
}
