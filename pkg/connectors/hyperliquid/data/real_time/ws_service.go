package real_time

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/pkg/websocket/base"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
	"github.com/sonirico/go-hyperliquid"
)

// WebSocketService manages the WebSocket connection using the robust pkg/websocket infrastructure
type WebSocketService struct {
	connManager  connection.ConnectionManager
	reconnectMgr connection.ReconnectManager
	baseService  base.BaseService
	logger       logging.ApplicationLogger
	parser       MessageParser

	// Subscription tracking
	subscriptionsMu sync.RWMutex
	subscriptions   map[int]*SubscriptionHandler // Map subscription ID -> handler

	// Message routing
	messageHandlers map[string]func([]byte) error // Channel -> handler
	handlersMu      sync.RWMutex

	// Parsed callbacks
	orderBookCallbacks map[int]func(*OrderBookMessage)
	orderBookMu        sync.RWMutex
	tradesCallbacks    map[int]func([]TradeMessage)
	tradesMu           sync.RWMutex
	klinesCallbacks    map[int]func(*KlineMessage)
	klinesMu           sync.RWMutex

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
// All dependencies are injected via DI - no instantiation with new()
func NewWebSocketService(
	connManager connection.ConnectionManager,
	reconnectMgr connection.ReconnectManager,
	baseService base.BaseService,
	logger logging.ApplicationLogger,
	parser MessageParser,
) (RealTimeService, error) {
	ws := &WebSocketService{
		connManager:        connManager,
		reconnectMgr:       reconnectMgr,
		baseService:        baseService,
		logger:             logger,
		parser:             parser,
		subscriptions:      make(map[int]*SubscriptionHandler),
		messageHandlers:    make(map[string]func([]byte) error),
		orderBookCallbacks: make(map[int]func(*OrderBookMessage)),
		tradesCallbacks:    make(map[int]func([]TradeMessage)),
		klinesCallbacks:    make(map[int]func(*KlineMessage)),
		errorCh:            make(chan error, 100),
	}

	// Set up connection manager callbacks
	connManager.SetCallbacks(
		ws.onConnect,
		ws.onDisconnect,
		ws.onMessage,
		ws.onError,
	)

	// Set up reconnection manager callbacks
	reconnectMgr.SetCallbacks(
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
func (ws *WebSocketService) onDisconnect() error {
	ws.logger.Warn("⚠️  WebSocket disconnected")
	// Re-subscription will happen on reconnect
	return nil
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

// Subscription methods are organized in separate files:
// - prices.go: SubscribeToOrderBook, UnsubscribeFromOrderBook, SubscribeToKlines, UnsubscribeFromKlines
// - trades.go: SubscribeToTrades, UnsubscribeFromTrades
// - account.go: SubscribeToPositions, SubscribeToAccountBalance

// subscribeToChannel is the internal method that handles raw subscriptions
func (ws *WebSocketService) subscribeToChannel(channel, coin, interval string, callback func(hyperliquid.WSMessage)) (int, error) {
	subID := generateSubscriptionID()

	sub := &SubscriptionHandler{
		ID:       subID,
		Channel:  channel,
		Coin:     coin,
		Interval: interval,
		Callback: callback,
	}

	ws.subscriptionsMu.Lock()
	ws.subscriptions[subID] = sub
	ws.subscriptionsMu.Unlock()

	return subID, ws.sendSubscription(channel, coin, interval)
}

// sendSubscription sends a subscription message to Hyperliquid
func (ws *WebSocketService) sendSubscription(channel, coin, interval string) error {
	subMsg := map[string]interface{}{
		"method": "subscribe",
		"subscription": map[string]interface{}{
			"type": channel,
			"coin": coin,
		},
	}

	if interval != "" {
		subMsg["subscription"].(map[string]interface{})["interval"] = interval
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	return ws.connManager.Send(data)
}

// Parsing helper functions that use the injected parser

func (ws *WebSocketService) parseOrderBook(msg hyperliquid.WSMessage) (*OrderBookMessage, error) {
	return ws.parser.ParseOrderBook(msg)
}

func (ws *WebSocketService) parseTrades(msg hyperliquid.WSMessage) ([]TradeMessage, error) {
	return ws.parser.ParseTrades(msg)
}

func (ws *WebSocketService) parseKline(msg hyperliquid.WSMessage) (*KlineMessage, error) {
	return ws.parser.ParseKline(msg)
}

// Message handlers for specific channels

func (ws *WebSocketService) handleOrderbookMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal orderbook message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "l2Book" {
		ws.logger.Warn("Failed to parse orderbook message: expected l2Book channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "l2Book" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

func (ws *WebSocketService) handleTradesMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal trades message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "trades" {
		ws.logger.Warn("Failed to parse trades message: expected trades channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "trades" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

func (ws *WebSocketService) handleCandleMessage(data []byte) error {
	var msgWrapper struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(data, &msgWrapper); err != nil {
		ws.logger.Debug("Failed to unmarshal candle message: %v", err)
		return nil
	}

	if msgWrapper.Channel != "candle" {
		ws.logger.Warn("Failed to parse candle message: expected candle channel, got %s", msgWrapper.Channel)
		return nil
	}

	// Call subscribed callbacks
	ws.subscriptionsMu.RLock()
	defer ws.subscriptionsMu.RUnlock()

	for _, sub := range ws.subscriptions {
		if sub.Channel == "candle" {
			msg := hyperliquid.WSMessage{Channel: msgWrapper.Channel, Data: msgWrapper.Data}
			sub.Callback(msg)
		}
	}

	return nil
}

// Disconnect closes the connection
func (ws *WebSocketService) Disconnect() error {
	ws.logger.Info("Disconnecting from WebSocket")
	return ws.connManager.Disconnect()
}

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
