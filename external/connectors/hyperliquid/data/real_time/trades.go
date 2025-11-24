package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToTrades(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToTrades(coin, callback)
}

func (r *realTimeService) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "trades", Coin: coin}
	return ws.Unsubscribe(sub, subscriptionID)
}
