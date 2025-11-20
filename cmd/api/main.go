package main

import (
	"github.com/backtesting-org/kronos-sdk/kronos"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	externalConnectors "github.com/backtesting-org/live-trading/external/connectors"
	"github.com/backtesting-org/live-trading/internal/api"
	"github.com/backtesting-org/live-trading/internal/config"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/infrastructure"
	"github.com/backtesting-org/live-trading/internal/services"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// === Core Dependencies ===
		fx.Provide(config.LoadConfig),

		// === Kronos SDK (includes registry, stores, analytics, etc.) ===
		kronos.Module,

		// === Infrastructure (logging, lifecycle) ===
		infrastructure.Module,

		// === Database ===
		database.Module,

		// === Services ===
		services.Module,

		// === Exchange Configs ===
		exchange.Module,

		// === Connectors (auto-register to SDK registry) ===
		externalConnectors.Module,

		// === HTTP API (handlers, routes, server) ===
		api.Module,
	).Run()
}
