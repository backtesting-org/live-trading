package real_time

import "github.com/sonirico/go-hyperliquid"

func (w *RealTimeData) SubscribeToTrades(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToTrades(coin, callback)
}

func (w *RealTimeData) UnsubscribeFromTrades(coin string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "trades", Coin: coin}
	return w.ws.Unsubscribe(sub, subscriptionID)
}
