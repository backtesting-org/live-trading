package pkg

import (
	"github.com/backtesting-org/live-trading/pkg/connectors"
	"github.com/backtesting-org/live-trading/pkg/runtime"
	"go.uber.org/fx"
)

var Module = fx.Provide(
	connectors.Module,
	runtime.Module,
)
