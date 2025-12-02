package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error) {
	// Wrap the user callback with parsing logic and error handling
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseOrderBook(msg)
		if err != nil {
			r.logger.Warn("Failed to parse orderbook message: %v", err)
			return
		}
		if parsed.Coin == coin {
			callback(parsed)
		}
	}

	subID, err := r.ws.SubscribeToOrderbook(coin, wrappedCallback)
	if err != nil {
		r.logger.Error("Failed to subscribe to orderbook %s: %v", coin, err)
		return 0, fmt.Errorf("failed to subscribe to orderbook: %w", err)
	}

	r.logger.Info("Successfully subscribed to orderbook: %s (ID: %d)", coin, subID)
	return subID, nil
}

func (r *realTimeService) SubscribeToKlines(coin, interval string, callback func(*KlineMessage)) (int, error) {
	// Wrap the user callback with parsing logic and error handling
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseKline(msg)
		if err != nil {
			r.logger.Warn("Failed to parse kline message: %v", err)
			return
		}
		if parsed.Coin == coin {
			callback(parsed)
		}
	}

	subID, err := r.ws.SubscribeToCandles(coin, interval, wrappedCallback)
	if err != nil {
		r.logger.Error("Failed to subscribe to klines %s %s: %v", coin, interval, err)
		return 0, fmt.Errorf("failed to subscribe to klines: %w", err)
	}

	r.logger.Info("Successfully subscribed to klines: %s %s (ID: %d)", coin, interval, subID)
	return subID, nil
}

func (r *realTimeService) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "l2Book", Coin: coin}
	if err := r.ws.Unsubscribe(sub, subscriptionID); err != nil {
		r.logger.Error("Failed to unsubscribe from orderbook %s: %v", coin, err)
		return fmt.Errorf("failed to unsubscribe from orderbook: %w", err)
	}
	r.logger.Info("Successfully unsubscribed from orderbook: %s", coin)
	return nil
}

func (r *realTimeService) UnsubscribeFromKlines(coin, interval string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "candle", Coin: coin, Interval: interval}
	if err := r.ws.Unsubscribe(sub, subscriptionID); err != nil {
		r.logger.Error("Failed to unsubscribe from klines %s %s: %v", coin, interval, err)
		return fmt.Errorf("failed to unsubscribe from klines: %w", err)
	}
	r.logger.Info("Successfully unsubscribed from klines: %s %s", coin, interval)
	return nil
}
