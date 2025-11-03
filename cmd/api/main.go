package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/backtesting-org/live-trading/internal/api"
	"github.com/backtesting-org/live-trading/internal/api/handlers"
	"github.com/backtesting-org/live-trading/internal/api/websocket"
	"github.com/backtesting-org/live-trading/internal/config"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/services"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting live-trading API server",
		zap.String("version", "1.0.0"),
		zap.Int("port", cfg.Server.Port))

	// Initialize database
	logger.Info("Connecting to database...")
	repo, err := database.NewRepository(cfg.Database.ConnectionString)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer repo.Close()

	// Run migrations
	ctx := context.Background()
	logger.Info("Running database migrations...")
	if err := runMigrations(ctx, repo); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Initialize services
	eventBus := services.NewEventBus()
	defer eventBus.Close()

	pluginManager := services.NewPluginManager(repo, logger, cfg.Plugin.Directory)
	strategyExecutor := services.NewStrategyExecutor(repo, pluginManager, logger, eventBus)

	// Initialize handlers
	pluginHandler := handlers.NewPluginHandler(pluginManager, logger, cfg.Server.MaxUploadSize)
	strategyHandler := handlers.NewStrategyHandler(strategyExecutor, logger)
	wsHandler := websocket.NewHandler(eventBus, logger)

	// Start WebSocket event listener
	wsHandler.StartEventListener()

	// Setup router
	router := api.SetupRouter(
		pluginHandler,
		strategyHandler,
		wsHandler,
		logger,
		cfg.Server.CORSAllowOrigin,
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server started",
			zap.String("address", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// initLogger initializes the zap logger
func initLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	var level zapcore.Level
	switch cfg.Level {
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
		Encoding:    cfg.Format,
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
		OutputPaths:      []string{cfg.OutputPath},
		ErrorOutputPaths: []string{"stderr"},
	}

	return logConfig.Build()
}

// runMigrations runs database migrations
func runMigrations(ctx context.Context, repo *database.Repository) error {
	// Read migration file
	migrationSQL, err := os.ReadFile("internal/database/migrations/001_initial_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	return repo.RunMigrations(ctx, string(migrationSQL))
}
