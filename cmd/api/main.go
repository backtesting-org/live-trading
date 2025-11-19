package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/stores/market"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/adaptor"
	"github.com/backtesting-org/live-trading/internal/api"
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"github.com/backtesting-org/live-trading/internal/api/websocket"
	"github.com/backtesting-org/live-trading/internal/config"
	"github.com/backtesting-org/live-trading/internal/connectors"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,
			provideLogger,
			provideRepository,
			provideParadexConfig,
			provideConnectorRegistry,
			provideConnector,
			provideTimeProvider,
			provideStore,
			provideTradingLogger,
			services.NewEventBus,
			services.NewPluginManager,
			services.NewPositionManager,
			services.NewKronosProvider,
			provideMarketDataFeed,
			provideTradeExecutor,
			services.NewStrategyExecutor,
			handlers.NewPluginHandler,
			handlers.NewStrategyHandler,
			handlers.NewOrdersHandler,
			handlers.NewDashboardHandler,
			websocket.NewHandler,
			provideHTTPServer,
		),
		fx.Invoke(
			runMigrations,
			startWebSocketListener,
			registerLifecycle,
		),
	)

	app.Run()
}

func provideLogger(cfg *config.Config) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.Logging.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	logConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false,
		Encoding:    cfg.Logging.Format,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{cfg.Logging.OutputPath},
		ErrorOutputPaths: []string{"stderr"},
	}

	return logConfig.Build()
}

func provideRepository(cfg *config.Config, logger *zap.Logger) (*database.Repository, error) {
	logger.Info("Connecting to database...")
	repo, err := database.NewRepository(cfg.Database.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return repo, nil
}

func provideParadexConfig(logger *zap.Logger) (*exchange.Paradex, error) {
	logger.Info("Loading Paradex configuration...")
	cfg := &exchange.Paradex{}

	// Try to load config, but don't fail if credentials are missing
	// This allows the server to start even without Paradex credentials
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("Paradex configuration incomplete - trading will be disabled until credentials are provided",
				zap.Any("error", r))
		}
	}()

	cfg.LoadParadexConfig()
	return cfg, nil
}

func provideConnectorRegistry(
	cfg *config.Config,
	paradexConfig *exchange.Paradex,
	tradingLogger logging.TradingLogger,
	logger *zap.Logger,
	timeProvider temporal.TimeProvider,
) *connectors.Registry {
	logger.Info("Initializing connector registry...")

	registry := connectors.NewRegistry()
	appLogger := services.NewApplicationLogger(logger)

	// Register Paradex connector provider (exchange-specific code in external/)
	registry.Register("paradex", func() (connector.Connector, error) {
		return paradex.NewConnector(paradexConfig, appLogger, tradingLogger, timeProvider)
	})

	// Future exchanges can be registered here:
	// registry.Register("binance", func() (connector.Connector, error) {
	//     return binance.NewConnector(binanceConfig, appLogger, tradingLogger, timeProvider)
	// })

	logger.Info("Connector registry initialized", zap.Strings("exchanges", registry.ListExchanges()))
	return registry
}

func provideConnector(
	cfg *config.Config,
	registry *connectors.Registry,
	logger *zap.Logger,
) (connector.Connector, error) {
	exchangeName := strings.ToLower(cfg.Exchange.Name)
	logger.Info("Creating exchange connector...", zap.String("exchange", exchangeName))

	conn, err := registry.GetConnector(exchangeName)
	if err != nil {
		logger.Warn("Failed to create connector - trading will be disabled", zap.Error(err))
		// Return nil connector instead of failing - allows server to start
		return nil, nil
	}

	// Onboard account if needed (Paradex-specific, but done in main.go for now)
	if exchangeName == "paradex" && conn != nil {
		logger.Info("Onboarding Paradex account...")
		if err := onboardParadexAccount(logger); err != nil {
			logger.Warn("Failed to onboard account", zap.Error(err))
		}
	}

	logger.Info("Connector initialized successfully", zap.String("exchange", exchangeName))
	return conn, nil
}

// onboardParadexAccount handles Paradex-specific onboarding
// TODO: Move this to external/exchanges/paradex as a hook/callback
func onboardParadexAccount(logger *zap.Logger) error {
	// Load Paradex config
	cfg := &exchange.Paradex{}
	cfg.LoadParadexConfig()

	appLogger := services.NewApplicationLogger(logger)
	client, err := adaptor.NewClient(cfg, appLogger)
	if err != nil {
		return err
	}

	if err := client.Onboard(context.Background()); err != nil {
		if strings.Contains(err.Error(), "ALREADY_ONBOARDED") || strings.Contains(err.Error(), "already onboarded") {
			logger.Info("Paradex account already onboarded")
			return nil
		}
		return err
	}

	logger.Info("Paradex account onboarded successfully")
	return nil
}

func provideTimeProvider() temporal.TimeProvider {
	return services.NewLiveTimeProvider()
}

func provideMarketDataFeed(
	conn connector.Connector,
	store market.MarketData,
	cfg *config.Config,
	logger *zap.Logger,
	timeProvider temporal.TimeProvider,
) *services.MarketDataFeed {
	// Map config exchange name to connector.ExchangeName
	var exchangeName connector.ExchangeName
	switch strings.ToLower(cfg.Exchange.Name) {
	case "paradex":
		exchangeName = connector.Paradex
	default:
		exchangeName = connector.Paradex
	}
	return services.NewMarketDataFeed(conn, store, logger, exchangeName, timeProvider)
}

func provideTradeExecutor(
	conn connector.Connector,
	positionManager *services.PositionManager,
	repo *database.Repository,
	eventBus *services.EventBus,
	logger *zap.Logger,
	timeProvider temporal.TimeProvider,
) *services.TradeExecutor {
	return services.NewTradeExecutor(conn, positionManager, repo, eventBus, logger, timeProvider)
}

func provideStore(timeProvider temporal.TimeProvider) market.MarketData {
	return services.NewMemoryStore(timeProvider)
}

func provideTradingLogger(logger *zap.Logger) logging.TradingLogger {
	return services.NewTradingLogger(logger)
}

func provideHTTPServer(
	cfg *config.Config,
	pluginHandler *handlers.PluginHandler,
	strategyHandler *handlers.StrategyHandler,
	ordersHandler *handlers.OrdersHandler,
	dashboardHandler *handlers.DashboardHandler,
	wsHandler *websocket.Handler,
	logger *zap.Logger,
	timeProvider temporal.TimeProvider,
) *http.Server {
	router := api.SetupRouter(
		pluginHandler,
		strategyHandler,
		ordersHandler,
		dashboardHandler,
		wsHandler,
		logger,
		cfg.Server.CORSAllowOrigin,
		timeProvider,
	)

	return &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}
}

func runMigrations(repo *database.Repository, logger *zap.Logger) error {
	logger.Info("Running database migrations...")

	migrationSQL, err := os.ReadFile("internal/database/migrations/001_initial_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	ctx := context.Background()
	if err := repo.RunMigrations(ctx, string(migrationSQL)); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Migrations completed successfully")
	return nil
}

func startWebSocketListener(wsHandler *websocket.Handler) {
	wsHandler.StartEventListener()
}

func registerLifecycle(
	lc fx.Lifecycle,
	server *http.Server,
	repo *database.Repository,
	eventBus *services.EventBus,
	marketDataFeed *services.MarketDataFeed,
	logger *zap.Logger,
	cfg *config.Config,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Start market data feed
			logger.Info("Starting market data feed...")
			if err := marketDataFeed.Start(); err != nil {
				logger.Error("Failed to start market data feed", zap.Error(err))
				// Don't fail startup if market feed fails - may not have Paradex credentials yet
			} else {
				logger.Info("Market data feed started successfully")
			}

			// Start HTTP server
			go func() {
				logger.Info("Server started",
					zap.String("address", server.Addr),
					zap.String("version", "1.0.0"))

				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Fatal("Failed to start server", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down server...")

			shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			// Stop market data feed
			logger.Info("Stopping market data feed...")
			marketDataFeed.Stop()

			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Error("Server forced to shutdown", zap.Error(err))
			}

			eventBus.Close()

			if err := repo.Close(); err != nil {
				logger.Error("Failed to close database connection", zap.Error(err))
			}

			logger.Info("Server stopped")
			return nil
		},
	})
}
