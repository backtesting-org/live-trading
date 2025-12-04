package websocket

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

// SubscribeToOrderBook subscribes to orderbook updates for a coin
func (ws *WebSocketService) SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	// Store the parsed callback
	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[subID] = callback
	ws.orderBookMu.Unlock()

	// Subscribe to raw message with parsing wrapper
	rawSubID, err := ws.subscribeToChannel("l2Book", coin, "", func(msg hyperliquid.WSMessage) {
		parsed, err := ws.parseOrderBook(msg)
		if err != nil {
			select {
			case ws.errorCh <- fmt.Errorf("failed to parse orderbook for %s: %w", coin, err):
			default:
			}
			return
		}

		ws.orderBookMu.RLock()
		cb, exists := ws.orderBookCallbacks[subID]
		ws.orderBookMu.RUnlock()

		if exists && cb != nil {
			cb(parsed)
		}
	})

	if err != nil {
		ws.orderBookMu.Lock()
		delete(ws.orderBookCallbacks, subID)
		ws.orderBookMu.Unlock()
		return 0, err
	}

	// Map parsed ID to raw ID for unsubscribe
	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	ws.subscriptionsMu.Unlock()

	return subID, nil
}

// UnsubscribeFromOrderBook unsubscribes from orderbook updates
func (ws *WebSocketService) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	ws.orderBookMu.Lock()
	delete(ws.orderBookCallbacks, subscriptionID)
	ws.orderBookMu.Unlock()

	// Find and remove the subscription
	ws.subscriptionsMu.Lock()
	for rawID, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "l2Book" && sub.Coin == coin {
			delete(ws.subscriptions, rawID)
			ws.subscriptionsMu.Unlock()
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}

// SubscribeToKlines subscribes to kline updates
func (ws *WebSocketService) SubscribeToKlines(coin, interval string, callback func(*KlineMessage)) (int, error) {
	if callback == nil {
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()

	ws.klinesMu.Lock()
	ws.klinesCallbacks[subID] = callback
	ws.klinesMu.Unlock()

	rawSubID, err := ws.subscribeToChannel("candle", coin, interval, func(msg hyperliquid.WSMessage) {
		parsed, err := ws.parseKline(msg)
		if err != nil {
			select {
			case ws.errorCh <- fmt.Errorf("failed to parse kline for %s %s: %w", coin, interval, err):
			default:
			}
			return
		}

		ws.klinesMu.RLock()
		cb, exists := ws.klinesCallbacks[subID]
		ws.klinesMu.RUnlock()

		if exists && cb != nil {
			cb(parsed)
		}
	})

	if err != nil {
		ws.klinesMu.Lock()
		delete(ws.klinesCallbacks, subID)
		ws.klinesMu.Unlock()
		return 0, err
	}

	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	ws.subscriptionsMu.Unlock()

	return subID, nil
}

// UnsubscribeFromKlines unsubscribes from kline updates
func (ws *WebSocketService) UnsubscribeFromKlines(coin, interval string, subscriptionID int) error {
	ws.klinesMu.Lock()
	delete(ws.klinesCallbacks, subscriptionID)
	ws.klinesMu.Unlock()

	ws.subscriptionsMu.Lock()
	for rawID, sub := range ws.subscriptions {
		if sub.ID == subscriptionID && sub.Channel == "candle" && sub.Coin == coin && sub.Interval == interval {
			delete(ws.subscriptions, rawID)
			ws.subscriptionsMu.Unlock()
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}
