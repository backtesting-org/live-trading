package clients

import (
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/sonirico/go-hyperliquid"
)

func NewInfoClient(cfg *exchange.HyperliquidConfig) *hyperliquid.Info {
	return hyperliquid.NewInfo(cfg.BaseURL, true, nil, nil)
}
