package main

import (
	"github.com/backtesting-org/kronos-sdk/kronos"
	"github.com/backtesting-org/live-trading/external/connectors"
	"github.com/backtesting-org/live-trading/internal/cli"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Kronos SDK (provides registry and other services)
		kronos.Module,

		// External connectors (registers with registry)
		connectors.Module,

		// CLI commands
		cli.Module,
	).Run()
}
