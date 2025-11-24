package real_time

import "github.com/sonirico/go-hyperliquid"

func (w *RealTimeData) SubscribeToUserEvents(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToUserEvents(user, callback)
}

func (w *RealTimeData) SubscribeToUserFills(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToUserFills(user, callback)
}

func (w *RealTimeData) SubscribeToUserFundings(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToUserFundings(user, callback)
}

func (w *RealTimeData) SubscribeToOrderUpdates(callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToOrderUpdates(callback)
}

func (w *RealTimeData) SubscribeToWebData(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToWebData2(user, callback)
}

func (w *RealTimeData) SubscribeToLedgerUpdates(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	return w.ws.SubscribeToUserNonFundingLedgerUpdates(user, callback)
}

func (w *RealTimeData) UnsubscribeFromUserEvents(user string, subscriptionID int) error {
	sub := hyperliquid.Subscription{Type: "userEvents", User: user}
	return w.ws.Unsubscribe(sub, subscriptionID)
}
