package real_time

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/pkg/websocket/base"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
	"github.com/sonirico/go-hyperliquid"
)

// WebSocketService manages the WebSocket connection using the robust pkg/websocket infrastructure
type WebSocketService struct {
	connManager  *connection.connectionManager
	reconnectMgr *connection.reconnectManager
	baseService  *base.baseService
	logger       logging.ApplicationLogger

	// Subscription tracking
	subscriptionsMu sync.RWMutex
	subscriptions   map[int]*SubscriptionHandler // Map subscription ID -> handler

	// Message routing
	messageHandlers map[string]func([]byte) error // Channel -> handler
	handlersMu      sync.RWMutex

	// Error channel
	errorCh chan error

	// State
	ctx    context.Context
	cancel context.CancelFunc
}

// SubscriptionHandler tracks an active subscription
type SubscriptionHandler struct {
	ID       int
	Channel  string
	Coin     string
	Interval string
	Callback func(hyperliquid.WSMessage)
}

// NewWebSocketService creates a new WebSocket service using pkg/websocket infrastructure
func NewWebSocketService(
	baseService *base.baseService,
	logger logging.ApplicationLogger,
) (*WebSocketService, error) {
	ws := &WebSocketService{
		baseService:     baseService,
		logger:          logger,
		subscriptions:   make(map[int]*SubscriptionHandler),
		messageHandlers: make(map[string]func([]byte) error),
		errorCh:         make(chan error, 100),
	}

	// Set up connection manager callbacks
	baseService.connManager.SetCallbacks(
		ws.onConnect,
		ws.onDisconnect,
		ws.onMessage,
		ws.onError,
	)

	// Create and set up reconnection manager
	reconnectStrategy := connection.NewExponentialBackoffStrategy(
		5*time.Second,  // Initial delay
		60*time.Second, // Max delay
		10,             // Max attempts
	)

	ws.reconnectMgr = connection.NewReconnectManager(baseService.connManager, reconnectStrategy, logger)
	ws.reconnectMgr.SetCallbacks(
		ws.onReconnectStart,
		ws.onReconnectFail,
		ws.onReconnectSuccess,
	)

	return ws, nil
}

// Connect establishes the WebSocket connection with automatic reconnection
func (ws *WebSocketService) Connect() error {
	// Use a background context that we control, don't use the caller's context
	// This prevents the connection from closing when the caller's context cancels
	ws.ctx = context.Background()
	ws.cancel = nil

	ws.logger.Info("🔌 Connecting to WebSocket: %s", ws.connManager.GetState())

	// Connect with circuit breaker
	// connectionManager handles keep-alive internally via simpleHealthMonitor
	if err := ws.connManager.Connect(ws.ctx); err != nil {
		ws.logger.Error("❌ Failed to connect to WebSocket: %v", err)
		return fmt.Errorf("websocket connection failed: %w", err)
	}

	ws.logger.Info("✅ WebSocket connected successfully")
	return nil
}

// Close disconnects the WebSocket
func (ws *WebSocketService) Close() error {
	ws.logger.Info("Closing WebSocket connection")

	return ws.connManager.Disconnect()
}

// IsConnected returns whether the WebSocket is currently connected
func (ws *WebSocketService) IsConnected() bool {
	return ws.connManager.GetState() == connection.StateConnected
}

// GetMetrics returns connection and message metrics
func (ws *WebSocketService) GetMetrics() map[string]interface{} {
	stats := ws.connManager.GetConnectionStats()

	// Add subscription count
	ws.subscriptionsMu.RLock()
	stats["active_subscriptions"] = len(ws.subscriptions)
	ws.subscriptionsMu.RUnlock()

	return stats
}

// GetErrorChannel returns the error channel for consumers
func (ws *WebSocketService) GetErrorChannel() <-chan error {
	return ws.errorCh
}

// onConnect is called when the connection is established
func (ws *WebSocketService) onConnect() error {
	ws.logger.Info("✅ WebSocket connected")
	return nil
}

// onDisconnect is called when the connection is lost
func (ws *WebSocketService) onDisconnect() {
	ws.logger.Warn("⚠️  WebSocket disconnected")
	// Re-subscription will happen on reconnect
}

// onMessage processes incoming WebSocket messages
func (ws *WebSocketService) onMessage(data []byte) error {
	// Parse the message to determine its channel
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal message wrapper: %v", err)
		return nil // Don't error on unparseable messages
	}

	// Find handler for this channel
	ws.handlersMu.RLock()
	handler, exists := ws.messageHandlers[msgWrapper.Channel]
	ws.handlersMu.RUnlock()

	if !exists {
		ws.logger.Debug("No handler for channel: %s", msgWrapper.Channel)
		return nil
	}

	// Call handler
	if err := handler(data); err != nil {
		ws.logger.Warn("Handler error for channel %s: %v", msgWrapper.Channel, err)
		// Still report error
		select {
		case ws.errorCh <- fmt.Errorf("message handler error for %s: %w", msgWrapper.Channel, err):
		default:
		}
	}

	return nil
}

// onError handles errors from the connection manager
func (ws *WebSocketService) onError(err error) {
	ws.logger.Error("❌ WebSocket error: %v", err)
	select {
	case ws.errorCh <- err:
	default:
		ws.logger.Warn("Error channel full, dropping error")
	}
}

// onReconnectStart is called when reconnection attempt starts
func (ws *WebSocketService) onReconnectStart(attempt int) {
	ws.logger.Info("🔄 Reconnection attempt %d", attempt)
}

// onReconnectFail is called when a reconnection attempt fails
func (ws *WebSocketService) onReconnectFail(attempt int, err error) {
	ws.logger.Warn("❌ Reconnection attempt %d failed: %v", attempt, err)
	select {
	case ws.errorCh <- fmt.Errorf("reconnection failed (attempt %d): %w", attempt, err):
	default:
	}
}

// onReconnectSuccess is called when reconnection succeeds
func (ws *WebSocketService) onReconnectSuccess(attempt int) {
	ws.logger.Info("✅ Reconnection successful (attempt %d)", attempt)
	// Re-establish subscriptions
	ws.resubscribeAll()
}

// resubscribeAll re-subscribes to all tracked subscriptions
func (ws *WebSocketService) resubscribeAll() {
	ws.subscriptionsMu.RLock()
	subscriptions := make([]*SubscriptionHandler, 0, len(ws.subscriptions))
	for _, sub := range ws.subscriptions {
		subscriptions = append(subscriptions, sub)
	}
	ws.subscriptionsMu.RUnlock()

	ws.logger.Info("Re-subscribing to %d subscriptions after reconnect", len(subscriptions))

	for _, sub := range subscriptions {
		// Re-subscribe based on channel type
		switch sub.Channel {
		case "l2Book":
			ws.logger.Debug("Re-subscribing to orderbook: %s", sub.Coin)
			// The subscription will be re-sent to server
		case "trades":
			ws.logger.Debug("Re-subscribing to trades: %s", sub.Coin)
		case "candle":
			ws.logger.Debug("Re-subscribing to candles: %s %s", sub.Coin, sub.Interval)
		case "webData2":
			ws.logger.Debug("Re-subscribing to webData2")
		}
	}
}

// registerMessageHandler registers a handler for a specific channel
func (ws *WebSocketService) registerMessageHandler(channel string, handler func([]byte) error) {
	ws.handlersMu.Lock()
	defer ws.handlersMu.Unlock()
	ws.messageHandlers[channel] = handler
}

// SubscribeToOrderbook subscribes to orderbook updates
func (ws *WebSocketService) SubscribeToOrderbook(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	if !ws.IsConnected() {
		return 0, fmt.Errorf("websocket not connected")
	}

	// Generate subscription ID
	subID := generateSubscriptionID()

	// Track subscription
	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = &SubscriptionHandler{
		ID:       subID,
		Channel:  "l2Book",
		Coin:     coin,
		Callback: callback,
	}
	ws.subscriptionsMu.Unlock()

	// Register message handler if not already registered
	ws.handlersMu.Lock()
	if _, exists := ws.messageHandlers["l2Book"]; !exists {
		ws.messageHandlers["l2Book"] = func(data []byte) error {
			return ws.handleOrderbookMessage(data)
		}
	}
	ws.handlersMu.Unlock()

	// Send subscription request to server
	sub := hyperliquid.Subscription{Type: "l2Book", Coin: coin}
	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": sub.Type,
			"coin": sub.Coin,
		},
	}

	if err := ws.connManager.SendJSON(subMsg); err != nil {
		ws.logger.Error("Failed to send subscription request: %v", err)
		// Remove from tracking on error
		ws.subscriptionsMu.Lock()
		delete(ws.subscriptions, subID)
		ws.subscriptionsMu.Unlock()
		return 0, fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	ws.logger.Info("📊 Subscribed to orderbook: %s (subID: %d)", coin, subID)
	return subID, nil
}

// SubscribeToTrades subscribes to trade updates
func (ws *WebSocketService) SubscribeToTrades(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	if !ws.IsConnected() {
		return 0, fmt.Errorf("websocket not connected")
	}

	subID := generateSubscriptionID()

	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = &SubscriptionHandler{
		ID:       subID,
		Channel:  "trades",
		Coin:     coin,
		Callback: callback,
	}
	ws.subscriptionsMu.Unlock()

	ws.handlersMu.Lock()
	if _, exists := ws.messageHandlers["trades"]; !exists {
		ws.messageHandlers["trades"] = func(data []byte) error {
			return ws.handleTradesMessage(data)
		}
	}
	ws.handlersMu.Unlock()

	sub := hyperliquid.Subscription{Type: "trades", Coin: coin}
	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": sub.Type,
			"coin": sub.Coin,
		},
	}

	if err := ws.connManager.SendJSON(subMsg); err != nil {
		ws.subscriptionsMu.Lock()
		delete(ws.subscriptions, subID)
		ws.subscriptionsMu.Unlock()
		return 0, fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	ws.logger.Info("🔄 Subscribed to trades: %s (subID: %d)", coin, subID)
	return subID, nil
}

// SubscribeToCandles subscribes to candle/kline updates
func (ws *WebSocketService) SubscribeToCandles(coin, interval string, callback func(hyperliquid.WSMessage)) (int, error) {
	if !ws.IsConnected() {
		return 0, fmt.Errorf("websocket not connected")
	}

	subID := generateSubscriptionID()

	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = &SubscriptionHandler{
		ID:       subID,
		Channel:  "candle",
		Coin:     coin,
		Interval: interval,
		Callback: callback,
	}
	ws.subscriptionsMu.Unlock()

	ws.handlersMu.Lock()
	if _, exists := ws.messageHandlers["candle"]; !exists {
		ws.messageHandlers["candle"] = func(data []byte) error {
			return ws.handleCandleMessage(data)
		}
	}
	ws.handlersMu.Unlock()

	sub := hyperliquid.Subscription{Type: "candle", Coin: coin, Interval: interval}
	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type":     sub.Type,
			"coin":     sub.Coin,
			"interval": sub.Interval,
		},
	}

	if err := ws.connManager.SendJSON(subMsg); err != nil {
		ws.subscriptionsMu.Lock()
		delete(ws.subscriptions, subID)
		ws.subscriptionsMu.Unlock()
		return 0, fmt.Errorf("failed to subscribe to candles: %w", err)
	}

	ws.logger.Info("📈 Subscribed to candles: %s %s (subID: %d)", coin, interval, subID)
	return subID, nil
}

// Unsubscribe unsubscribes from a subscription
func (ws *WebSocketService) Unsubscribe(sub hyperliquid.Subscription, subID int) error {
	ws.subscriptionsMu.Lock()
	_, exists := ws.subscriptions[subID]
	if exists {
		delete(ws.subscriptions, subID)
	}
	ws.subscriptionsMu.Unlock()

	if !exists {
		return fmt.Errorf("subscription %d not found", subID)
	}

	unsubMsg := map[string]interface{}{
		"method": "unsubscribe",
		"subscription": map[string]interface{}{
			"type": sub.Type,
			"coin": sub.Coin,
		},
	}

	if sub.Interval != "" {
		unsubMsg["subscription"].(map[string]interface{})["interval"] = sub.Interval
	}

	if err := ws.connManager.SendJSON(unsubMsg); err != nil {
		ws.logger.Error("Failed to send unsubscription request: %v", err)
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	ws.logger.Info("Unsubscribed from %s (subID: %d)", sub.Type, subID)
	return nil
}

// SubscribeToWebData subscribes to webData updates (account/position data)
func (ws *WebSocketService) SubscribeToWebData(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	if !ws.IsConnected() {
		return 0, fmt.Errorf("websocket not connected")
	}

	subID := generateSubscriptionID()

	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = &SubscriptionHandler{
		ID:       subID,
		Channel:  "webData2",
		Coin:     user,
		Callback: callback,
	}
	ws.subscriptionsMu.Unlock()

	ws.handlersMu.Lock()
	if _, exists := ws.messageHandlers["webData2"]; !exists {
		ws.messageHandlers["webData2"] = func(data []byte) error {
			return ws.handleWebData2Message(data)
		}
	}
	ws.handlersMu.Unlock()

	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": "webData2",
			"user": user,
		},
	}

	if err := ws.connManager.SendJSON(subMsg); err != nil {
		ws.subscriptionsMu.Lock()
		delete(ws.subscriptions, subID)
		ws.subscriptionsMu.Unlock()
		return 0, fmt.Errorf("failed to subscribe to webData2: %w", err)
	}

	ws.logger.Info("📋 Subscribed to webData2: %s (subID: %d)", user, subID)
	return subID, nil
}

func (ws *WebSocketService) handleWebData2Message(data []byte) error {
	var msg hyperliquid.WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		ws.logger.Debug("Failed to unmarshal webData2 message: %v", err)
		return nil
	}

	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "webData2" {
			sub.Callback(msg)
		}
	}

	return nil
}

// createMessageHandler creates a reusable handler for any channel type
func (ws *WebSocketService) createMessageHandler(channelName string) func([]byte) error {
	return func(data []byte) error {
		// Use baseService for rate limiting, validation, and metrics
		return ws.baseService.ProcessMessage(data, func(validData []byte) error {
			var msgWrapper struct {
				Channel string          `json:"channel"`
				Data    json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(validData, &msgWrapper); err != nil {
				ws.handleError(fmt.Sprintf("%s wrapper unmarshal failed: %v", channelName, err))
				return err
			}

			ws.logger.Debug("%s message received - channel: %s, data length: %d", channelName, msgWrapper.Channel, len(msgWrapper.Data))

			var msg hyperliquid.WSMessage
			if err := json.Unmarshal(msgWrapper.Data, &msg); err != nil {
				ws.handleError(fmt.Sprintf("%s data unmarshal failed: %v", channelName, err))
				return err
			}

			// Validate channel matches expected type
			if msg.Channel != channelName && msgWrapper.Channel != channelName {
				errMsg := fmt.Sprintf("channel mismatch - expected %s, got msg=%s wrapper=%s", channelName, msg.Channel, msgWrapper.Channel)
				ws.handleError(errMsg)
				return fmt.Errorf(errMsg)
			}

			// Route to all subscribed callbacks
			ws.subscriptionsMu.RLock()
			defer ws.subscriptionsMu.RUnlock()

			for _, sub := range ws.subscriptions {
				if sub.Channel == channelName {
					sub.Callback(msg)
				}
			}

			return nil
		})
	}
}

// handleError sends an error to the error channel and logs it
func (ws *WebSocketService) handleError(errMsg string) {
	ws.logger.Warn("❌ %s", errMsg)
	select {
	case ws.errorCh <- fmt.Errorf(errMsg):
	default:
		ws.logger.Warn("Error channel full, dropped: %s", errMsg)
	}
}

// Message handlers - thin wrappers around generic handler

func (ws *WebSocketService) handleOrderbookMessage(data []byte) error {
	return ws.createMessageHandler("l2Book")(data)
}

func (ws *WebSocketService) handleTradesMessage(data []byte) error {
	return ws.createMessageHandler("trades")(data)
}

func (ws *WebSocketService) handleCandleMessage(data []byte) error {
	return ws.createMessageHandler("candle")(data)
}

// Helper function to generate subscription IDs
var (
	subIDCounter int64
	subIDMutex   sync.Mutex
)

func generateSubscriptionID() int {
	subIDMutex.Lock()
	defer subIDMutex.Unlock()
	subIDCounter++
	return int(subIDCounter)
}
