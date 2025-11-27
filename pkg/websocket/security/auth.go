package security

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// AuthProvider defines the interface for WebSocket authentication
type AuthProvider interface {
	GetAuthHeaders(ctx context.Context) (http.Header, error)
	IsAuthenticated() bool
	Refresh(ctx context.Context) error
	GetTokenExpiry() time.Time
}

// AuthManager handles authentication for WebSocket connections
type AuthManager struct {
	provider     AuthProvider
	refreshMutex sync.Mutex
	logger       Logger
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

func NewAuthManager(provider AuthProvider, logger Logger) *AuthManager {
	return &AuthManager{
		provider: provider,
		logger:   logger,
	}
}

func (am *AuthManager) GetSecureHeaders(ctx context.Context) (http.Header, error) {
	// Ensure authentication is valid
	if !am.provider.IsAuthenticated() {
		if err := am.refreshAuth(ctx); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Check if token is expiring soon (within 5 minutes)
	if time.Until(am.provider.GetTokenExpiry()) < 5*time.Minute {
		am.logger.Debug("Token expiring soon, refreshing authentication")
		if err := am.refreshAuth(ctx); err != nil {
			am.logger.Warn("Failed to refresh expiring token: %v", err)
		}
	}

	headers, err := am.provider.GetAuthHeaders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth headers: %w", err)
	}

	// Add security headers
	if headers == nil {
		headers = make(http.Header)
	}

	headers.Set("User-Agent", "Trading-Bot-WebSocket/1.0")

	return headers, nil
}

func (am *AuthManager) refreshAuth(ctx context.Context) error {
	am.refreshMutex.Lock()
	defer am.refreshMutex.Unlock()

	// Double-check authentication status after acquiring lock
	if am.provider.IsAuthenticated() {
		return nil
	}

	am.logger.Debug("Refreshing authentication")
	if err := am.provider.Refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh authentication: %w", err)
	}

	am.logger.Debug("Authentication refreshed successfully")
	return nil
}

func (am *AuthManager) ValidateConnection(ctx context.Context) error {
	if !am.provider.IsAuthenticated() {
		return fmt.Errorf("not authenticated")
	}

	// Check token expiry
	if time.Now().After(am.provider.GetTokenExpiry()) {
		return fmt.Errorf("authentication token expired")
	}

	return nil
}

// PeriodicRefresh starts a goroutine that periodically refreshes authentication
func (am *AuthManager) PeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := am.refreshAuth(ctx); err != nil {
				am.logger.Error("Periodic auth refresh failed: %v", err)
			}
		}
	}
}
