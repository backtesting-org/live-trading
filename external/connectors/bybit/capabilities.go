package bybit

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/shopspring/decimal"
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
		Name:             connector.Bybit,
		TradingEnabled:   b.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      decimal.NewFromFloat(125.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USDT",
	}
}
