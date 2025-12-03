package security

import (
	"context"
	"net/http"
	"time"
)

// AuthProvider defines the interface for WebSocket authentication
type AuthProvider interface {
	GetAuthHeaders(ctx context.Context) (http.Header, error)
	IsAuthenticated() bool
	Refresh(ctx context.Context) error
	GetTokenExpiry() time.Time
}

// AuthManager defines authentication management operations
type AuthManager interface {
	GetSecureHeaders(ctx context.Context) (http.Header, error)
	ValidateConnection(ctx context.Context) error
	PeriodicRefresh(ctx context.Context, interval time.Duration)
}

// RateLimiter defines rate limiting operations
type RateLimiter interface {
	Allow() bool
	Reset()
}

// MessageValidator defines message validation operations
type MessageValidator interface {
	ValidateMessage(message []byte) error
}
