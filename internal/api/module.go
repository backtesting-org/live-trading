package api

import (
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"go.uber.org/fx"
)

// Module provides HTTP API components (handlers, routes, server)
var Module = fx.Module("api",
	fx.Provide(
		// HTTP Handlers
		//handlers.NewPluginHandler,
		//handlers.NewStrategyHandler,
		handlers.NewOrdersHandler,
		//handlers.NewDashboardHandler,
		//websocket.NewHandler,

		// HTTP Server
		//NewHTTPServer,
	),
	fx.Invoke(
		// Start websocket listener
		//func(wsHandler *websocket.Handler) {
		//	wsHandler.StartEventListener()
		//},
	),
)
