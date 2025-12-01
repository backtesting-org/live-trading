package types

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
)

const (
	Hyperliquid connector.ExchangeName = "hyperliquid"
	Paradex     connector.ExchangeName = "paradex"
	Bybit       connector.ExchangeName = "bybit"
)

// ConnectorInfo contains metadata about an available connector
type ConnectorInfo struct {
	ExchangeName  connector.ExchangeName
	ConfigExample connector.Config
}
