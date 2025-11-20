package infrastructure

import (
	"context"

	"github.com/backtesting-org/live-trading/internal/database"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RegisterLifecycle sets up application startup and shutdown hooks
func RegisterLifecycle(
	lc fx.Lifecycle,
	repo *database.Repository,
	logger *zap.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down server...")

			if err := repo.Close(); err != nil {
				logger.Error("Failed to close database connection", zap.Error(err))
			}

			logger.Info("Server stopped")
			return nil
		},
	})
}
