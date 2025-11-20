package services

import (
	"go.uber.org/fx"
)

// Module provides application services
var Module = fx.Module("services",
	fx.Provide(
		NewEventBus,
		NewPluginManager,
		NewLiveTimeProvider,
		NewMemoryStore,
		NewTradingLogger,
		NewApplicationLogger,
	),
)
