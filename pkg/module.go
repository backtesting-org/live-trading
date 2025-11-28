package pkg

import (
	"github.com/backtesting-org/kronos-sdk/kronos"
	"github.com/backtesting-org/live-trading/pkg/connectors"
	"github.com/backtesting-org/live-trading/pkg/runtime"
	"go.uber.org/fx"
)

var Module = fx.Options(
	kronos.Module,
	connectors.Module,
	runtime.Module,
)
