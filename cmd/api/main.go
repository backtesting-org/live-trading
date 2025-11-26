package main

import (
	"github.com/backtesting-org/kronos-sdk/kronos"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	externalConnectors "github.com/backtesting-org/live-trading/external/connectors"
	"github.com/backtesting-org/live-trading/internal/database"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// === Kronos SDK (includes registry, stores, analytics, etc.) ===
		kronos.Module,

		// === Infrastructure (logging, lifecycle) ===
		//infrastructure.Module,

		// === Database ===
		database.Module,

		// === Services (Strategy Runner, Auto-Load Plugins) ===
		//services.Module,

		// === Exchange Configs ===
		exchange.Module,

		// === Connectors (auto-register to SDK registry) ===
		externalConnectors.Module,
	).Run()
}
