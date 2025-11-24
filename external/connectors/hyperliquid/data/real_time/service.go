package real_time

import (
	"context"
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/clients"
)

// RealTimeService interface for WebSocket subscriptions
type RealTimeService interface {
	Connect(ctx context.Context) error
	Disconnect() error

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
}

type realTimeService struct {
	client clients.WebSocketClient
	parser *Parser
}

func NewRealTimeService(client clients.WebSocketClient, logger logging.ApplicationLogger) RealTimeService {
	return &realTimeService{
		client: client,
		parser: NewParser(logger),
	}
}

func (r *realTimeService) Connect(ctx context.Context) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.Connect(ctx)
}

func (r *realTimeService) Disconnect() error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.Close()
}
