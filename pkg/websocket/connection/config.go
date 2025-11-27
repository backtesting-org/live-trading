package connection

import (
	"fmt"
	"time"
)

// Config holds WebSocket connection configuration
type Config struct {
	// Connection settings
	URL              string        `json:"url" validate:"required,url"`
	ConnectTimeout   time.Duration `json:"connect_timeout"`
	HandshakeTimeout time.Duration `json:"handshake_timeout"`

	// Buffer settings
	ReadBufferSize  int   `json:"read_buffer_size"`
	WriteBufferSize int   `json:"write_buffer_size"`
	MaxMessageSize  int64 `json:"max_message_size"`

	// Timing settings
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	PingInterval time.Duration `json:"ping_interval"`
	PongTimeout  time.Duration `json:"pong_timeout"`

	// Reconnection settings
	EnableReconnect bool          `json:"enable_reconnect"`
	ReconnectDelay  time.Duration `json:"reconnect_delay"`
	MaxReconnects   int           `json:"max_reconnects"`

	// Security settings
	RequireSSL    bool `json:"require_ssl"`
	SkipTLSVerify bool `json:"skip_tls_verify"`

	// Rate limiting
	RateLimitCapacity int           `json:"rate_limit_capacity"`
	RateLimitRefill   time.Duration `json:"rate_limit_refill"`

	// Performance settings
	EnableCompression bool `json:"enable_compression"`
	EnablePooling     bool `json:"enable_pooling"`

	EnableHealthMonitoring bool          `json:"enable_health_monitoring"`
	EnableHealthPings      bool          `json:"enable_health_pings"`
	HealthCheckInterval    time.Duration `json:"health_check_interval"`
	HealthCheckTimeout     time.Duration `json:"health_check_timeout"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		ConnectTimeout:         30 * time.Second,
		HandshakeTimeout:       45 * time.Second,
		ReadBufferSize:         4096,
		WriteBufferSize:        4096,
		MaxMessageSize:         1024 * 1024, // 1MB
		ReadTimeout:            60 * time.Second,
		WriteTimeout:           10 * time.Second,
		PingInterval:           30 * time.Second,
		PongTimeout:            10 * time.Second,
		EnableReconnect:        true,
		ReconnectDelay:         5 * time.Second,
		MaxReconnects:          10,
		RequireSSL:             true,
		SkipTLSVerify:          false,
		RateLimitCapacity:      1000,
		RateLimitRefill:        time.Second,
		EnableCompression:      false,
		EnablePooling:          true,
		EnableHealthMonitoring: true,
		EnableHealthPings:      true,
		HealthCheckInterval:    30 * time.Second,
		HealthCheckTimeout:     10 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("URL is required")
	}

	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect timeout must be positive")
	}

	if c.ReadBufferSize <= 0 {
		return fmt.Errorf("read buffer size must be positive")
	}

	if c.WriteBufferSize <= 0 {
		return fmt.Errorf("write buffer size must be positive")
	}

	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("max message size must be positive")
	}

	if c.EnableReconnect && c.MaxReconnects <= 0 {
		return fmt.Errorf("max reconnects must be positive when reconnection is enabled")
	}

	if c.EnableHealthMonitoring && c.HealthCheckInterval <= 0 {
		return fmt.Errorf("health check interval must be positive when health monitoring is enabled")
	}

	if c.EnableHealthMonitoring && c.HealthCheckTimeout <= 0 {
		return fmt.Errorf("health check timeout must be positive when health monitoring is enabled")
	}

	return nil
}

// ApplyDefaults fills in missing values with defaults
func (c *Config) ApplyDefaults() {
	defaults := DefaultConfig()

	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = defaults.ConnectTimeout
	}
	if c.HandshakeTimeout == 0 {
		c.HandshakeTimeout = defaults.HandshakeTimeout
	}
	if c.ReadBufferSize == 0 {
		c.ReadBufferSize = defaults.ReadBufferSize
	}
	if c.WriteBufferSize == 0 {
		c.WriteBufferSize = defaults.WriteBufferSize
	}
	if c.MaxMessageSize == 0 {
		c.MaxMessageSize = defaults.MaxMessageSize
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaults.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaults.WriteTimeout
	}
	if c.PingInterval == 0 {
		c.PingInterval = defaults.PingInterval
	}
	if c.PongTimeout == 0 {
		c.PongTimeout = defaults.PongTimeout
	}
	if c.ReconnectDelay == 0 {
		c.ReconnectDelay = defaults.ReconnectDelay
	}
	if c.MaxReconnects == 0 {
		c.MaxReconnects = defaults.MaxReconnects
	}
	if c.RateLimitCapacity == 0 {
		c.RateLimitCapacity = defaults.RateLimitCapacity
	}
	if c.RateLimitRefill == 0 {
		c.RateLimitRefill = defaults.RateLimitRefill
	}

	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = defaults.HealthCheckInterval
	}
	if c.HealthCheckTimeout == 0 {
		c.HealthCheckTimeout = defaults.HealthCheckTimeout
	}
}

// TradingConfig returns a configuration optimized for trading applications
func TradingConfig(url string) Config {
	config := DefaultConfig()
	config.URL = url
	config.PingInterval = 15 * time.Second  // More frequent pings for trading
	config.PongTimeout = 5 * time.Second    // Shorter timeout for faster detection
	config.RateLimitCapacity = 2000         // Higher capacity for trading data
	config.MaxMessageSize = 2 * 1024 * 1024 // 2MB for large orderbooks
	config.EnablePooling = true             // Enable pooling for performance
	return config
}

// TestConfig returns a configuration suitable for testing
func TestConfig(url string) Config {
	config := DefaultConfig()
	config.URL = url
	config.ConnectTimeout = 5 * time.Second
	config.HandshakeTimeout = 10 * time.Second
	config.PingInterval = 5 * time.Second
	config.PongTimeout = 2 * time.Second
	config.MaxReconnects = 3
	config.ReconnectDelay = time.Second
	config.RequireSSL = false // Allow non-SSL for testing
	return config
}
