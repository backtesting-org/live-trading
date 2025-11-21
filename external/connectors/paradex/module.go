package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/external/connectors/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/connectors/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/connectors/paradex/websocket"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		adaptor.NewClient,
		requests.NewService,
		websockets.NewService,
		NewParadex,
	),
	// Automatically register paradex with the SDK registry at startup
	fx.Invoke(registerParadex),
)

// registerParadex registers the paradex connector with the SDK's ConnectorRegistry
func registerParadex(paradexConn connector.Connector, reg registry.ConnectorRegistry) {
	// Register the connector so it's available globally in the SDK
	reg.RegisterConnector(connector.Paradex, paradexConn)
}
