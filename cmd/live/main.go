package main

import (
	"github.com/backtesting-org/kronos-sdk/kronos"
	"github.com/backtesting-org/live-trading/external/connectors"
	"github.com/backtesting-org/live-trading/internal/cli"
	"go.uber.org/fx"
)

func main() {
	// Parse CLI first - exits early if --help is used
	cliArgs := cli.ParseCLI()

	// If we get here, user wants to run a strategy - initialize fx
	if cliArgs != nil {
		fx.New(
			// Kronos SDK (provides registry and other services)
			kronos.Module,

			// External connectors (registers with registry)
			connectors.Module,

			// CLI module
			cli.Module,

			// Supply parsed CLI args
			fx.Supply(cliArgs),

			// Execute the strategy
			fx.Invoke(cli.ExecuteStrategy),
		).Run()
	}
}
