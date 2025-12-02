package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToTrades(coin string, callback func([]TradeMessage)) (int, error) {
	// Wrap the user callback with parsing logic and error handling
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseTrades(msg)
		if err != nil {
			r.logger.Warn("Failed to parse trades message: %v", err)
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

	subID, err := r.ws.SubscribeToTrades(coin, wrappedCallback)
	if err != nil {
		r.logger.Error("Failed to subscribe to trades %s: %v", coin, err)
		return 0, fmt.Errorf("failed to subscribe to trades: %w", err)
	}

	r.logger.Info("Successfully subscribed to trades: %s (ID: %d)", coin, subID)
	return subID, nil
}

func (r *realTimeService) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "trades", Coin: coin}
	if err := r.ws.Unsubscribe(sub, subscriptionID); err != nil {
		r.logger.Error("Failed to unsubscribe from trades %s: %v", coin, err)
		return fmt.Errorf("failed to unsubscribe from trades: %w", err)
	}
	r.logger.Info("Successfully unsubscribed from trades: %s", coin)
	return nil
}
