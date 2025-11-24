package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToTrades(coin string, callback func([]TradeMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}

	// Wrap the user callback with parsing logic
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseTrades(msg)
		if err != nil {
			// Parser already logs the error
			return
		}

		// Filter trades for the requested coin
		filtered := []TradeMessage{}
		for _, trade := range parsed {
			if trade.Coin == coin {
				filtered = append(filtered, trade)
			}
		}

		if len(filtered) > 0 {
			callback(filtered)
		}
	}

	return ws.SubscribeToTrades(coin, wrappedCallback)
}

func (r *realTimeService) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "trades", Coin: coin}
	return ws.Unsubscribe(sub, subscriptionID)
}
