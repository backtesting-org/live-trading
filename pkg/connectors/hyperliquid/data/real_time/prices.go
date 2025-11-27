package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}

	// Wrap the user callback with parsing logic
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseOrderBook(msg)
		if err != nil {
			// Parser already logs the error
			return
		}
		if parsed.Coin == coin {
			callback(parsed)
		}
	}

	return ws.SubscribeToOrderbook(coin, wrappedCallback)
}

func (r *realTimeService) SubscribeToKlines(coin, interval string, callback func(*KlineMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}

	// Wrap the user callback with parsing logic
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseKline(msg)
		if err != nil {
			// Parser already logs the error
			return
		}
		if parsed.Coin == coin {
			callback(parsed)
		}
	}

	return ws.SubscribeToCandles(coin, interval, wrappedCallback)
}

func (r *realTimeService) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "l2Book", Coin: coin}
	return ws.Unsubscribe(sub, subscriptionID)
}

func (r *realTimeService) UnsubscribeFromKlines(coin, interval string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "candle", Coin: coin, Interval: interval}
	return ws.Unsubscribe(sub, subscriptionID)
}
