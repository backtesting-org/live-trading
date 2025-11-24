package real_time

import (
	"context"
	"fmt"

	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/clients"
	"github.com/sonirico/go-hyperliquid"
)

// RealTimeService interface for WebSocket subscriptions
type RealTimeService interface {
	Connect(ctx context.Context) error
	Disconnect() error

	// Price subscriptions
	SubscribeToAllPrices(callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToOrderBook(coin string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToBestBidOffer(coin string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToCandles(coin, interval string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToActiveAssetContext(coin string, callback func(hyperliquid.WSMessage)) (int, error)
	UnsubscribeFromOrderBook(coin string, subscriptionID int) error

	// Trade subscriptions
	SubscribeToTrades(coin string, callback func(hyperliquid.WSMessage)) (int, error)
	UnsubscribeFromTrades(coin string, subscriptionID int) error

	// User subscriptions
	SubscribeToUserEvents(user string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToUserFills(user string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToUserFundings(user string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToOrderUpdates(callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToWebData(user string, callback func(hyperliquid.WSMessage)) (int, error)
	SubscribeToLedgerUpdates(user string, callback func(hyperliquid.WSMessage)) (int, error)
	UnsubscribeFromUserEvents(user string, subscriptionID int) error
}

type realTimeService struct {
	client clients.WebSocketClient
}

func NewRealTimeService(client clients.WebSocketClient) RealTimeService {
	return &realTimeService{client: client}
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
