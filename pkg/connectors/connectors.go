package connectors

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	localConnector "github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex"
)

// ConnectorInfo contains metadata about an available connector
type ConnectorInfo struct {
	ExchangeName  connector.ExchangeName
	ConfigExample localConnector.Config
}

// AvailableConnectors is a map of all available exchange connectors with their config types
var AvailableConnectors = map[connector.ExchangeName]localConnector.Config{
	connector.Paradex:     &paradex.Config{},
	connector.Hyperliquid: &hyperliquid.Config{},
	connector.Bybit:       &bybit.Config{},
}

// IsAvailable checks if a connector is available for the given exchange
func IsAvailable(exchange connector.ExchangeName) bool {
	_, exists := AvailableConnectors[exchange]
	return exists
}

// ListAvailable returns a list of all available exchange names
func ListAvailable() []connector.ExchangeName {
	exchanges := make([]connector.ExchangeName, 0, len(AvailableConnectors))
	for exchange := range AvailableConnectors {
		exchanges = append(exchanges, exchange)
	}
	return exchanges
}

// GetConfigType returns the config type for a given exchange
func GetConfigType(exchange connector.ExchangeName) localConnector.Config {
	return AvailableConnectors[exchange]
}
