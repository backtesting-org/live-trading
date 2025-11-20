package infrastructure

import (
	"context"
	"net/http"
	"time"

	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegisterLifecycle sets up application startup and shutdown hooks
func RegisterLifecycle(
	lc fx.Lifecycle,
	server *http.Server,
	repo *database.Repository,
	eventBus *services.EventBus,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
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
