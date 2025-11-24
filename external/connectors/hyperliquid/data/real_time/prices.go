package real_time

import "github.com/sonirico/go-hyperliquid"

func (w *RealTimeData) SubscribeToAllPrices(callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToAllMids(callback)
}

func (w *RealTimeData) SubscribeToOrderBook(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToOrderbook(coin, callback)
}

func (w *RealTimeData) SubscribeToBestBidOffer(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToBBO(coin, callback)
}

func (w *RealTimeData) SubscribeToCandles(coin, interval string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToCandles(coin, interval, callback)
}

func (w *RealTimeData) SubscribeToActiveAssetContext(coin string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToActiveAssetCtx(coin, callback)
}

func (w *RealTimeData) UnsubscribeFromOrderBook(coin string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "l2Book", Coin: coin}
	return w.ws.Unsubscribe(sub, subscriptionID)
}
