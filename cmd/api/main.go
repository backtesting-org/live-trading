package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/backtesting-org/live-trading/internal/api"
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"github.com/backtesting-org/live-trading/internal/api/websocket"
	"github.com/backtesting-org/live-trading/internal/config"
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
			services.NewEventBus,
			services.NewPluginManager,
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
	logger *zap.Logger,
	cfg *config.Config,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
