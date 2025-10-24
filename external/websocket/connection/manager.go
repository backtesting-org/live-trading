package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	performance2 "github.com/backtesting-org/live-trading/external/websocket/performance"
	"github.com/backtesting-org/live-trading/external/websocket/security"
	"github.com/gorilla/websocket"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateFailed
)

func (cs ConnectionState) String() string {
	switch cs {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ConnectionManager handles WebSocket connection lifecycle with standard ping/pong
type ConnectionManager struct {
	config         Config
	authManager    *security.AuthManager
	metrics        *performance2.Metrics
	circuitBreaker *performance2.CircuitBreaker
	logger         security.Logger

	conn       *websocket.Conn
	state      ConnectionState
	stateMutex sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc

	lastActivity  time.Time
	activityMutex sync.RWMutex

	onConnect    func() error
	onDisconnect func()
	onMessage    func([]byte) error
	onError      func(error)
}

func NewConnectionManager(
	config Config,
	authManager *security.AuthManager,
	metrics *performance2.Metrics,
	logger security.Logger,
) *ConnectionManager {
	return &ConnectionManager{
		config:         config,
		authManager:    authManager,
		metrics:        metrics,
		circuitBreaker: performance2.NewCircuitBreaker(3, 30*time.Second),
		logger:         logger,
		state:          StateDisconnected,
	}
}

func (cm *ConnectionManager) SetCallbacks(
	onConnect func() error,
	onDisconnect func(),
	onMessage func([]byte) error,
	onError func(error),
) {
	cm.onConnect = onConnect
	cm.onDisconnect = onDisconnect
	cm.onMessage = onMessage
	cm.onError = onError
}

func (cm *ConnectionManager) Connect(ctx context.Context) error {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	if cm.state == StateConnected || cm.state == StateConnecting {
		return fmt.Errorf("already connected or connecting")
	}

	cm.setState(StateConnecting)
	cm.ctx, cm.cancel = context.WithCancel(ctx)

	return cm.circuitBreaker.Call(func() error {
		return cm.doConnect()
	})
}

func (cm *ConnectionManager) doConnect() error {
	u, err := url.Parse(cm.config.URL)
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	if u.Scheme != "wss" {
		return fmt.Errorf("insecure WebSocket scheme: %s (must be wss)", u.Scheme)
	}

	headers, err := cm.authManager.GetSecureHeaders(cm.ctx)
	if err != nil {
		return fmt.Errorf("failed to get auth headers: %w", err)
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: cm.config.HandshakeTimeout,
		ReadBufferSize:   cm.config.ReadBufferSize,
		WriteBufferSize:  cm.config.WriteBufferSize,
	}

	connectCtx, cancel := context.WithTimeout(cm.ctx, cm.config.ConnectTimeout)
	defer cancel()

	conn, _, err := dialer.DialContext(connectCtx, u.String(), headers)
	if err != nil {
		cm.setState(StateFailed)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	// Set up standard WebSocket ping/pong handlers
	conn.SetReadLimit(cm.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(cm.config.ReadTimeout))

	// Handle pings from server - respond with pongs
	conn.SetPingHandler(func(appData string) error {
		cm.logger.Debug("📥 Received WebSocket ping from server")

		err := conn.WriteControl(websocket.PongMessage, []byte(appData),
			time.Now().Add(5*time.Second))

		if err != nil {
			cm.logger.Debug("Failed to send pong: %v", err)
		} else {
			cm.logger.Debug("📤 Sent WebSocket pong response")
		}
		return err
	})

	// Handle pongs from server
	conn.SetPongHandler(func(appData string) error {
		cm.logger.Debug("✅ Received WebSocket pong from server")
		cm.updateLastActivity()
		conn.SetReadDeadline(time.Now().Add(cm.config.ReadTimeout))
		return nil
	})

	cm.conn = conn
	cm.setState(StateConnected)
	cm.updateLastActivity()

	// Start core connection handlers
	go cm.readMessages()

	// Optional: Basic health monitoring (configurable)
	if cm.config.EnableHealthMonitoring {
		go cm.simpleHealthMonitor()
	}

	if cm.onConnect != nil {
		if err := cm.onConnect(); err != nil {
			cm.logger.Error("Connect callback failed: %v", err)
			return err
		}
	}

	cm.logger.Info("WebSocket connected successfully to %s", cm.config.URL)
	return nil
}

func (cm *ConnectionManager) Disconnect() error {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	if cm.state == StateDisconnected {
		return nil
	}

	cm.setState(StateDisconnected)

	if cm.cancel != nil {
		cm.cancel()
	}

	var err error
	if cm.conn != nil {
		err = cm.conn.Close()
		cm.conn = nil
	}

	if cm.onDisconnect != nil {
		cm.onDisconnect()
	}

	cm.logger.Info("WebSocket disconnected")
	return err
}

func (cm *ConnectionManager) SendMessage(message []byte) error {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	return cm.conn.WriteMessage(websocket.TextMessage, message)
}

func (cm *ConnectionManager) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	// Generic debug logging (not exchange-specific)
	cm.logger.Debug("Sending WebSocket message: %s", string(data))

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	return cm.conn.WriteJSON(v)
}

func (cm *ConnectionManager) SendPing() error {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	if cm.state != StateConnected || cm.conn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	if err := cm.conn.SetWriteDeadline(time.Now().Add(cm.config.WriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	cm.logger.Debug("Sending WebSocket ping control frame")
	return cm.conn.WriteMessage(websocket.PingMessage, nil)
}

func (cm *ConnectionManager) GetState() ConnectionState {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()
	return cm.state
}

func (cm *ConnectionManager) GetConnectionStats() map[string]interface{} {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	cm.activityMutex.RLock()
	lastActivity := cm.lastActivity
	cm.activityMutex.RUnlock()

	stats := map[string]interface{}{
		"state":         cm.state.String(),
		"connected":     cm.state == StateConnected,
		"last_activity": lastActivity,
		"url":           cm.config.URL,
	}

	if cm.metrics != nil {
		for k, v := range cm.metrics.GetStats() {
			stats[k] = v
		}
	}

	return stats
}

func (cm *ConnectionManager) IsHealthy() bool {
	if cm.GetState() != StateConnected {
		return false
	}

	cm.activityMutex.RLock()
	lastActivity := cm.lastActivity
	cm.activityMutex.RUnlock()

	// Consider connection healthy if we've had activity recently
	return time.Since(lastActivity) <= cm.config.HealthCheckTimeout
}

func (cm *ConnectionManager) setState(state ConnectionState) {
	cm.state = state
	cm.logger.Debug("Connection state changed to: %s", state.String())
}

func (cm *ConnectionManager) updateLastActivity() {
	cm.activityMutex.Lock()
	defer cm.activityMutex.Unlock()
	cm.lastActivity = time.Now()
}

func (cm *ConnectionManager) readMessages() {
	defer func() {
		if r := recover(); r != nil {
			cm.logger.Error("WebSocket read panic: %v", r)
			cm.handleConnectionError()
		}
	}()

	for {
		select {
		case <-cm.ctx.Done():
			cm.logger.Debug("WebSocket read loop cancelled by context")
			return
		default:
			if cm.GetState() != StateConnected {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if err := cm.conn.SetReadDeadline(time.Now().Add(cm.config.ReadTimeout)); err != nil {
				cm.logger.Error("Failed to set read deadline: %v", err)
				cm.handleConnectionError()
				return
			}

			_, message, err := cm.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					cm.logger.Info("WebSocket closed normally")
				} else {
					cm.logger.Error("WebSocket read error: %v", err)
				}

				cm.handleConnectionError()
				return
			}

			// Update activity on any message received
			cm.updateLastActivity()

			if cm.metrics != nil {
				cm.metrics.IncrementReceived()
			}

			if cm.onMessage != nil {
				if err := cm.onMessage(message); err != nil {
					cm.logger.Debug("Message handler error: %v", err)
					if cm.onError != nil {
						cm.onError(fmt.Errorf("message processing error: %w", err))
					}
				}
			}
		}
	}
}

// Optional simple health monitoring (non-aggressive)
func (cm *ConnectionManager) simpleHealthMonitor() {
	ticker := time.NewTicker(cm.config.HealthCheckInterval)
	defer ticker.Stop()

	cm.logger.Debug("Starting connection health monitor with %v interval", cm.config.HealthCheckInterval)

	for {
		select {
		case <-cm.ctx.Done():
			cm.logger.Debug("Health monitor cancelled by context")
			return
		case <-ticker.C:
			if cm.GetState() != StateConnected {
				cm.logger.Debug("Health check: not connected")
				return
			}

			cm.activityMutex.RLock()
			timeSinceActivity := time.Since(cm.lastActivity)
			cm.activityMutex.RUnlock()

			if timeSinceActivity > cm.config.HealthCheckTimeout {
				cm.logger.Warn("No activity for %v, connection may be stale", timeSinceActivity)

				// Optionally send a ping to test connection
				if cm.config.EnableHealthPings {
					if err := cm.SendPing(); err != nil {
						cm.logger.Debug("Health ping failed: %v", err)
						cm.handleConnectionError()
						return
					}
				}
			} else {
				cm.logger.Debug("Connection healthy: activity %v ago", timeSinceActivity)
			}
		}
	}
}

func (cm *ConnectionManager) handleConnectionError() {
	cm.stateMutex.Lock()
	previousState := cm.state
	cm.setState(StateDisconnected)
	cm.stateMutex.Unlock()

	cm.logger.Debug("WebSocket connection error detected, previous state: %s", previousState.String())

	if cm.conn != nil {
		cm.conn.Close()
		cm.conn = nil
	}

	if cm.metrics != nil {
		cm.metrics.IncrementConnectionError()
	}

	if cm.onDisconnect != nil {
		cm.onDisconnect()
	}

	if cm.onError != nil {
		cm.onError(fmt.Errorf("WebSocket connection lost"))
	}
}
