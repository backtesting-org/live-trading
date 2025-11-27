package hyperliquid

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/trading"
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
		fx.Annotate(
			NewHyperliquid,
			fx.ResultTags(`name:"hyperliquid"`),
		),
	),
	// Automatically register hyperliquid with the SDK registry at startup
	fx.Invoke(fx.Annotate(
		registerHyperliquid,
		fx.ParamTags(`name:"hyperliquid"`),
	)),
)

// registerHyperliquid registers the hyperliquid connector with the SDK's ConnectorRegistry
func registerHyperliquid(hyperliquidConn pkgconnector.Initializable, reg registry.ConnectorRegistry) {
	// Register the connector (Initializable embeds connector.Connector)
	reg.RegisterConnector(connector.Hyperliquid, hyperliquidConn)
}
