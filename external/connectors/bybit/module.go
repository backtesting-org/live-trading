package bybit

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/external/connectors/bybit/data"
	"github.com/backtesting-org/live-trading/external/connectors/bybit/data/real_time"
	"github.com/backtesting-org/live-trading/external/connectors/bybit/trading"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		trading.NewTradingService,
		data.NewMarketDataService,
		real_time.NewRealTimeService,
		NewBybit,
	),
	fx.Invoke(registerBybit),
)

func registerBybit(bybitConn pkgconnector.Initializable, reg registry.ConnectorRegistry) {
	reg.RegisterConnector(connector.Bybit, bybitConn)
}
