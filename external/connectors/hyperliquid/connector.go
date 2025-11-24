package hyperliquid

import (
	"context"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/trading"
)

// Hyperliquid implements the exchange.Exchange interface for Hyperliquid DEX
type Hyperliquid struct {
	marketData *data.MarketDataService
	trading    *trading.TradingService
	config     *exchange.HyperliquidConfig
	appLogger  logging.ApplicationLogger
	ctx        context.Context
}

// Reset implements connector.Connector interface
// For live exchanges, reset is a no-op since they don't maintain simulated state
func (h *Hyperliquid) Reset() error {
	// Live exchanges don't maintain internal simulation state to reset
	return nil
}

// Ensure Hyperliquid implements connector.Connector interface at compile time
var _ connector.Connector = (*Hyperliquid)(nil)

// NewHyperliquid creates a new Hyperliquid exchange instance
func NewHyperliquid(
	marketData *data.MarketDataService,
	trading *trading.TradingService,
	config *exchange.HyperliquidConfig,
	appLogger logging.ApplicationLogger,
) *Hyperliquid {
	return &Hyperliquid{
		marketData: marketData,
		trading:    trading,
		config:     config,
		appLogger:  appLogger,
		ctx:        context.Background(),
	}
}
