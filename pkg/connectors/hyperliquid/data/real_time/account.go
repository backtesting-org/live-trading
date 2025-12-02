package real_time

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (r *realTimeService) SubscribeToPositions(user string, callback func(*PositionMessage)) (int, error) {
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParsePosition(msg)
		if err != nil {
			r.logger.Warn("Failed to parse position message: %v", err)
			return
		}
		callback(parsed)
	}

	subID, err := r.ws.SubscribeToWebData(user, wrappedCallback)
	if err != nil {
		r.logger.Error("Failed to subscribe to positions for user %s: %v", user, err)
		return 0, fmt.Errorf("failed to subscribe to positions: %w", err)
	}

	r.logger.Info("Successfully subscribed to positions for user: %s (ID: %d)", user, subID)
	return subID, nil
}

func (r *realTimeService) SubscribeToAccountBalance(user string, callback func(*AccountBalanceMessage)) (int, error) {
	wrappedCallback := func(msg hyperliquid.WSMessage) {
		parsed, err := r.parser.ParseAccountBalance(msg)
		if err != nil {
			r.logger.Warn("Failed to parse account balance message: %v", err)
			return
		}
		callback(parsed)
	}

	subID, err := r.ws.SubscribeToWebData(user, wrappedCallback)
	if err != nil {
		r.logger.Error("Failed to subscribe to account balance for user %s: %v", user, err)
		return 0, fmt.Errorf("failed to subscribe to account balance: %w", err)
	}

	r.logger.Info("Successfully subscribed to account balance for user: %s (ID: %d)", user, subID)
	return subID, nil
}
