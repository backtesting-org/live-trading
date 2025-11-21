package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		NewParadex,
	),
	// Automatically register paradex with the SDK registry at startup
	fx.Invoke(registerParadex),
)

// registerParadex registers the paradex connector with the SDK's ConnectorRegistry
func registerParadex(paradexConn pkgconnector.Initializable, reg registry.ConnectorRegistry) {
	// Register the connector (Initializable embeds connector.Connector)
	reg.RegisterConnector(connector.Paradex, paradexConn)
}
