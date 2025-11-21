package connectors

import (
	"github.com/backtesting-org/live-trading/external/connectors/paradex"
	"go.uber.org/fx"
)

// Module includes all exchange connector modules
// Each connector module automatically registers itself via fx groups
var Module = fx.Options(
	paradex.Module,
)
