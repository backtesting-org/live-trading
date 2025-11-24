package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToPositions(user string, callback func(*PositionMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}

	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParsePosition(msg)
		if err != nil {
			return
		}
		callback(parsed)
	}

	return ws.SubscribeToWebData2(user, wrappedCallback)
}

func (r *realTimeService) SubscribeToAccountBalance(user string, callback func(*AccountBalanceMessage)) (int, error) {
	ws, err := r.client.GetWebSocket()
	if err != nil {
		return 0, fmt.Errorf("websocket not configured: %w", err)
	}

	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseAccountBalance(msg)
		if err != nil {
			return
		}
		callback(parsed)
	}

	return ws.SubscribeToWebData2(user, wrappedCallback)
}
