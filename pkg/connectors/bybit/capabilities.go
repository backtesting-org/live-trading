package bybit

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// SupportsTradingOperations returns whether trading operations are supported
func (b *bybit) SupportsTradingOperations() bool {
	return b.trading != nil
}

// SupportsRealTimeData returns whether real-time data is supported
func (b *bybit) SupportsRealTimeData() bool {
	return true
}

// SupportsHistoricalData returns whether historical data is supported
func (b *bybit) SupportsHistoricalData() bool {
	return b.marketData != nil
}

func (b *bybit) SupportsPerpetuals() bool {
	return true
}

func (b *bybit) SupportsSpot() bool {
	return true
}

// GetConnectorInfo returns metadata about the exchange
func (b *bybit) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             types.Bybit,
		TradingEnabled:   b.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      numerical.NewFromFloat(125.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USDT",
	}
}
