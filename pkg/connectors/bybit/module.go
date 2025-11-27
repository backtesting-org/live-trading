package bybit

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/trading"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		trading.NewTradingService,
		data.NewMarketDataService,
		real_time.NewRealTimeService,
		fx.Annotate(
			NewBybit,
			fx.ResultTags(`name:"bybit"`),
		),
	),
	fx.Invoke(fx.Annotate(
		registerBybit,
		fx.ParamTags(`name:"bybit"`),
	)),
)

func registerBybit(bybitConn pkgconnector.Initializable, reg registry.ConnectorRegistry) {
	reg.RegisterConnector(connector.Bybit, bybitConn)
}
