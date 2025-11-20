package api

// NewHTTPServer creates the HTTP server
//func NewHTTPServer(
//	cfg *config.Config,
//	pluginHandler *handlers.PluginHandler,
//	strategyHandler *handlers.StrategyHandler,
//	ordersHandler *handlers.OrdersHandler,
//	dashboardHandler *handlers.DashboardHandler,
//	wsHandler *websocket.Handler,
//	logger *zap.Logger,
//	timeProvider temporal.TimeProvider,
//) *http.Server {
//	router := SetupRouter(
//		pluginHandler,
//		strategyHandler,
//		ordersHandler,
//		dashboardHandler,
//		wsHandler,
//		logger,
//		cfg.Server.CORSAllowOrigin,
//		timeProvider,
//	)
//
//	return &http.Server{
//		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
//		Handler:      router,
//		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
//		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
//	}
//}
