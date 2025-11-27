package base

import (
	"context"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/pkg/websocket/performance"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
)

type Config struct {
	URL               string
	ReconnectDelay    time.Duration
	MaxReconnects     int
	PingInterval      time.Duration
	PongTimeout       time.Duration
	MaxMessageSize    int
	RateLimitCapacity int
	RateLimitRefill   time.Duration
}

type BaseService struct {
	// Configuration
	config Config
	logger logging.ApplicationLogger

	// Security components
	rateLimiter *security.RateLimiter
	validator   *security.MessageValidator

	// Performance components
	metrics        *performance.Metrics
	circuitBreaker *performance.CircuitBreaker

	// Connection state
	isConnected bool
	connMutex   sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewBaseService(config Config, logger logging.ApplicationLogger) *BaseService {
	// Create validation config
	validationConfig := security.ValidationConfig{
		MaxMessageSize: config.MaxMessageSize,
		AllowedTypes: map[string]bool{
			"pong":         true,
			"subscription": true,
			"update":       true,
			"error":        true,
		},
	}

	return &BaseService{
		config:         config,
		logger:         logger,
		rateLimiter:    security.NewRateLimiter(config.RateLimitCapacity, config.RateLimitRefill),
		validator:      security.NewMessageValidator(validationConfig),
		metrics:        performance.NewMetrics(),
		circuitBreaker: performance.NewCircuitBreaker(3, 30*time.Second),
	}
}

func (bs *BaseService) ProcessMessage(message []byte, handler func([]byte) error) error {
	start := time.Now()
	bs.metrics.IncrementReceived()

	defer func() {
		latency := time.Since(start)
		bs.metrics.IncrementProcessed(latency)

		if latency > 10*time.Millisecond {
			bs.logger.Warn("Slow message processing: %v", latency)
		}
	}()

	// Rate limiting
	if !bs.rateLimiter.Allow() {
		bs.metrics.IncrementDropped()
		bs.logger.Warn("Message rate limit exceeded, dropping message")
		return nil
	}

	// Validation
	if err := bs.validator.ValidateMessage(message); err != nil {
		bs.metrics.IncrementDropped()
		bs.logger.Warn("Message validation failed: %v", err)
		return nil
	}

	// Process message
	return handler(message)
}

func (bs *BaseService) GetMetrics() map[string]interface{} {
	return bs.metrics.GetStats()
}

func (bs *BaseService) IsConnected() bool {
	bs.connMutex.RLock()
	defer bs.connMutex.RUnlock()
	return bs.isConnected
}

func (bs *BaseService) SetConnected(connected bool) {
	bs.connMutex.Lock()
	defer bs.connMutex.Unlock()
	bs.isConnected = connected
}
