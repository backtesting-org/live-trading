package real_time

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
)

// RealTimeService interface for WebSocket subscriptions
type RealTimeService interface {
	Connect() error
	Disconnect() error
	GetErrorChannel() <-chan error

	// Price subscriptions - callbacks receive parsed message types
	SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error)
	UnsubscribeFromOrderBook(coin string, subscriptionID int) error

	// Trade subscriptions - callbacks receive parsed message types
	SubscribeToTrades(coin string, callback func([]TradeMessage)) (int, error)
	UnsubscribeFromTrades(coin string, subscriptionID int) error

	// User subscriptions - callbacks receive parsed message types
	SubscribeToPositions(user string, callback func(*PositionMessage)) (int, error)
	SubscribeToAccountBalance(user string, callback func(*AccountBalanceMessage)) (int, error)
	SubscribeToKlines(coin, interval string, callback func(*KlineMessage)) (int, error)
	UnsubscribeFromKlines(coin, interval string, subscriptionID int) error
}

type realTimeService struct {
	ws     *WebSocketService
	parser *Parser
	logger logging.ApplicationLogger
}

func NewRealTimeService(
	wsURL string,
	authManager *security.authManager,
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) (RealTimeService, error) {
	// Create the new WebSocket service using robust infrastructure
	ws, err := NewWebSocketService(wsURL, authManager, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create websocket service: %w", err)
	}

	return &realTimeService{
		ws:     ws,
		parser: NewParser(logger, timeProvider),
		logger: logger,
	}, nil
}

func (r *realTimeService) Connect() error {
	r.logger.Info("Connecting to WebSocket for real-time data")
	return r.ws.Connect()
}

func (r *realTimeService) Disconnect() error {
	r.logger.Info("Disconnecting from WebSocket")
	return r.ws.Close()
}

func (r *realTimeService) GetErrorChannel() <-chan error {
	return r.ws.GetErrorChannel()
}
