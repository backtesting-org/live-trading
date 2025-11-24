package hyperliquid

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/trading"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		clients.NewExchangeClient,
		clients.NewInfoClient,
		clients.NewWebSocketClient,
		trading.NewTradingService,
		data.NewMarketDataService,
		real_time.NewRealTimeService,
		NewHyperliquid,
	),
	// Automatically register hyperliquid with the SDK registry at startup
	fx.Invoke(registerHyperliquid),
)

// registerHyperliquid registers the hyperliquid connector with the SDK's ConnectorRegistry
func registerHyperliquid(hyperliquidConn pkgconnector.Initializable, reg registry.ConnectorRegistry) {
	// Register the connector (Initializable embeds connector.Connector)
	reg.RegisterConnector(connector.Hyperliquid, hyperliquidConn)
}
