package hyperliquid

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/shopspring/decimal"
)

// SupportsTradingOperations returns whether trading operations are supported
func (h *Hyperliquid) SupportsTradingOperations() bool {
	return h.trading != nil
}

// SupportsRealTimeData returns whether real-time data is supported
func (h *Hyperliquid) SupportsRealTimeData() bool {
	return true
}

// SupportsHistoricalData returns whether historical data is supported
func (h *Hyperliquid) SupportsHistoricalData() bool {
	return h.marketData != nil
}

func (h *Hyperliquid) SupportsPerpetuals() bool {
	return true
}

// turned off as their spot market is weird
func (h *Hyperliquid) SupportsSpot() bool {
	return false
}

// GetConnectorInfo returns metadata about the exchange
func (h *Hyperliquid) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             connector.Hyperliquid,
		TradingEnabled:   h.SupportsTradingOperations(),
		WebSocketEnabled: true,
		MaxLeverage:      decimal.NewFromFloat(50.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USD",
	}
}
