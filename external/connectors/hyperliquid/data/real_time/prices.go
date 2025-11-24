package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToAllPrices(callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToAllMids(callback)
}

func (r *realTimeService) SubscribeToOrderBook(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToOrderbook(coin, callback)
}

func (r *realTimeService) SubscribeToBestBidOffer(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToBBO(coin, callback)
}

func (r *realTimeService) SubscribeToCandles(coin, interval string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToCandles(coin, interval, callback)
}

func (r *realTimeService) SubscribeToActiveAssetContext(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToActiveAssetCtx(coin, callback)
}

func (r *realTimeService) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "l2Book", Coin: coin}
	return ws.Unsubscribe(sub, subscriptionID)
}
