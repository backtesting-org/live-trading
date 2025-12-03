package hyperliquid

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/trading"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"go.uber.org/fx"
)

// Module is the main Hyperliquid connector module
var Module = fx.Options(
	real_time.WebSocketModule,

	fx.Provide(
		clients.NewExchangeClient,
		clients.NewInfoClient,
		clients.NewWebSocketClient,
		trading.NewTradingService,
		data.NewMarketDataService,
		fx.Annotate(
			NewHyperliquid,
			fx.ResultTags(`name:"hyperliquid"`),
		),
	),

	fx.Invoke(fx.Annotate(
		registerHyperliquid,
		fx.ParamTags(`name:"hyperliquid"`),
	)),
)

// registerHyperliquid registers the hyperliquid connector with the SDK's ConnectorRegistry
func registerHyperliquid(hyperliquidConn connector.Connector, reg registry.ConnectorRegistry) {
	reg.RegisterConnector(types.Hyperliquid, hyperliquidConn)
}
