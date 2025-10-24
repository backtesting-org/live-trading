package paradex

import (
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/exchanges/paradex/websocket"
	"go.uber.org/fx"
)

var ParadexModule = fx.Options(
	fx.Provide(
		adaptor.NewClient,
		requests.NewService,
		websockets.NewService,
	),
)
