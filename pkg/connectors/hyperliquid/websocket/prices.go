package websocket

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

// SubscribeToOrderBook subscribes to orderbook updates for a coin
func (ws *WebSocketService) SubscribeToOrderBook(coin string, callback func(*OrderBookMessage)) (int, error) {
	fmt.Printf("🟢 SubscribeToOrderBook CALLED for coin=%s\n", coin)

	if callback == nil {
		fmt.Printf("🔴 callback is NIL!\n")
		return 0, fmt.Errorf("callback cannot be nil")
	}

	subID := generateSubscriptionID()
	fmt.Printf("🟢 Generated subID=%d for orderbook %s\n", subID, coin)

	// Store the parsed callback
	ws.orderBookMu.Lock()
	ws.orderBookCallbacks[subID] = callback
	fmt.Printf("🟢 Stored callback in orderBookCallbacks[%d] (total: %d)\n", subID, len(ws.orderBookCallbacks))
	ws.orderBookMu.Unlock()

	// Subscribe to raw message with parsing wrapper
	fmt.Printf("🟢 About to call subscribeToChannel for l2Book/%s\n", coin)
	rawSubID, err := ws.subscribeToChannel("l2Book", coin, "", func(msg hyperliquid.WSMessage) {
		fmt.Printf("🔵 RAW CALLBACK INVOKED for l2Book/%s (rawSubID will be set later)\n", coin)

		parsed, err := ws.parseOrderBook(msg)
		if err != nil {
			fmt.Printf("🔴 PARSE ERROR: %v\n", err)
			ws.logger.Warn("Failed to parse orderbook: %v", err)
			return
		}

		fmt.Printf("🟢 PARSED SUCCESSFULLY: %+v\n", parsed)

		ws.orderBookMu.RLock()
		cb, exists := ws.orderBookCallbacks[subID]
		ws.orderBookMu.RUnlock()

		fmt.Printf("🔍 Looking up callback for subID=%d: exists=%v, cb!=nil=%v\n", subID, exists, cb != nil)

		if exists && cb != nil {
			fmt.Printf("🎯 Calling user callback for subID=%d\n", subID)
			cb(parsed)
			fmt.Printf("✅ User callback completed for subID=%d\n", subID)
		} else {
			fmt.Printf("🔴 NO CALLBACK FOUND for subID=%d\n", subID)
		}
	})

	if err != nil {
		fmt.Printf("🔴 subscribeToChannel FAILED: %v\n", err)
		ws.orderBookMu.Lock()
		delete(ws.orderBookCallbacks, subID)
		ws.orderBookMu.Unlock()
		return 0, err
	}

	fmt.Printf("🟢 subscribeToChannel returned rawSubID=%d\n", rawSubID)

	// Map parsed ID to raw ID for unsubscribe
	ws.subscriptionsMu.Lock()
	ws.subscriptions[rawSubID].ID = subID
	fmt.Printf("🟢 Mapped rawSubID=%d to subID=%d\n", rawSubID, subID)
	ws.subscriptionsMu.Unlock()

	ws.logger.Info("✅ Subscribed to orderbook for %s (ID: %d)", coin, subID)
	fmt.Printf("✅ SubscribeToOrderBook SUCCESS for %s: subID=%d\n", coin, subID)
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
			ws.logger.Info("Unsubscribed from orderbook for %s (ID: %d)", coin, subscriptionID)
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
			ws.logger.Warn("Failed to parse kline: %v", err)
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

	ws.logger.Info("📈 Subscribed to klines for %s %s (ID: %d)", coin, interval, subID)
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
			ws.logger.Info("Unsubscribed from klines for %s %s (ID: %d)", coin, interval, subscriptionID)
			return nil
		}
	}
	ws.subscriptionsMu.Unlock()

	return fmt.Errorf("subscription not found")
}
