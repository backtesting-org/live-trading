package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store"
	"github.com/backtesting-org/live-trading/internal/api"
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"github.com/backtesting-org/live-trading/internal/api/websocket"
	"github.com/backtesting-org/live-trading/internal/config"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/exchanges/paradex"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/exchanges/paradex/websocket"
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
			provideParadexConnector,
			provideStore,
			provideTradingLogger,
			services.NewEventBus,
			services.NewPluginManager,
			services.NewPositionManager,
			services.NewKronosProvider,
			services.NewMarketDataFeed,
			services.NewTradeExecutor,
			services.NewStrategyExecutor,
			handlers.NewPluginHandler,
			handlers.NewStrategyHandler,
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

func provideParadexConnector(
	cfg *exchange.Paradex,
	tradingLogger logging.TradingLogger,
	logger *zap.Logger,
) (*paradex.Paradex, error) {
	logger.Info("Initializing Paradex connector...")

	// Create application logger adapter for Paradex
	appLogger := services.NewApplicationLogger(logger)

	// Create HTTP client
	client, err := adaptor.NewClient(cfg, appLogger)
	if err != nil {
		logger.Warn("Failed to create Paradex client - trading will be disabled", zap.Error(err))
		// Return nil connector instead of failing - allows server to start
		return nil, nil
	}

	// Create Paradex services
	requestsService := requests.NewService(client, appLogger)
	wsService := websockets.NewService(client, cfg, appLogger, tradingLogger)

	return paradex.NewParadex(requestsService, wsService, cfg, appLogger, tradingLogger), nil
}

func provideStore() store.Store {
	return services.NewMemoryStore()
}

func provideTradingLogger(logger *zap.Logger) logging.TradingLogger {
	return services.NewTradingLogger(logger)
}

func provideHTTPServer(
	cfg *config.Config,
	pluginHandler *handlers.PluginHandler,
	strategyHandler *handlers.StrategyHandler,
	wsHandler *websocket.Handler,
	logger *zap.Logger,
) *http.Server {
	router := api.SetupRouter(
		pluginHandler,
		strategyHandler,
		wsHandler,
		logger,
		cfg.Server.CORSAllowOrigin,
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
