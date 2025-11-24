package clients

import (
	"fmt"

	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/sonirico/go-hyperliquid"
)

func NewWebSocketClient(cfg *exchange.HyperliquidConfig) (*hyperliquid.WebsocketClient, error) {
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("private key required for websocket client")
	}
	return hyperliquid.NewWebsocketClient(cfg.BaseURL), nil
}
