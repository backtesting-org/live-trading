package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToUserEvents(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToUserEvents(user, callback)
}

func (r *realTimeService) SubscribeToUserFills(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToUserFills(user, callback)
}

func (r *realTimeService) SubscribeToUserFundings(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToUserFundings(user, callback)
}

func (r *realTimeService) SubscribeToOrderUpdates(callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToOrderUpdates(callback)
}

func (r *realTimeService) SubscribeToWebData(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToWebData2(user, callback)
}

func (r *realTimeService) SubscribeToLedgerUpdates(user string, callback func(hyperliquid.WSMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}
	return ws.SubscribeToUserNonFundingLedgerUpdates(user, callback)
}

func (r *realTimeService) UnsubscribeFromUserEvents(user string, subscriptionID int) error {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return fmt.Errorf("websocket not configured: %w", err)
	}
	sub := hyperliquid.Subscription{Type: "userEvents", User: user}
	return ws.Unsubscribe(sub, subscriptionID)
}
