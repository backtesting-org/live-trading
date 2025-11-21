package database

import (
	"context"
	"fmt"
	"os"

	"github.com/backtesting-org/live-trading/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides database connectivity and migrations
var Module = fx.Module("database",
	fx.Provide(ProvideRepository),
	fx.Invoke(runMigrations),
)

// ProvideRepository creates a database repository from config
func ProvideRepository(cfg *config.Config, logger *zap.Logger) (*Repository, error) {
	logger.Info("Connecting to database...")
	repo, err := NewRepository(cfg.Database.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return repo, nil
}

// runMigrations runs database migrations on startup
func runMigrations(repo *Repository, logger *zap.Logger) error {
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
