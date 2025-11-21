package services

import (
	"context"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/live-trading/internal/config"
	"github.com/backtesting-org/live-trading/internal/database"
	"go.uber.org/fx"
)

// Module provides all services
var Module = fx.Module("services",
	fx.Provide(
		NewPluginConfig,
		NewPluginStorage,
		NewStrategyRunner,
	),
	fx.Invoke(
		AutoLoadPlugins,
	),
)

// NewPluginStorage creates a plugin storage adapter using the database repository
func NewPluginStorage(repo *database.Repository) plugin.Storage {
	return newPluginStorageAdapter(repo)
}

// NewPluginConfig creates a plugin configuration
func NewPluginConfig(
	storage plugin.Storage,
	logger logging.ApplicationLogger,
	cfg *config.Config,
) plugin.Config {
	return plugin.Config{
		Storage:   storage,
		Logger:    logger,
		PluginDir: cfg.Plugin.Directory,
	}
}

// AutoLoadPlugins automatically loads and runs plugins marked for auto-start
func AutoLoadPlugins(
	lc fx.Lifecycle,
	runner *StrategyRunner,
	repo *database.Repository,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Get all plugins from database
			plugins, err := repo.ListPlugins(ctx, 100, 0)
			if err != nil {
				runner.logger.Error("Failed to list plugins for auto-load", "error", err.Error())
				return nil // Don't fail startup
			}

			if len(plugins) == 0 {
				runner.logger.Info("No plugins found to auto-load")
				return nil
			}

			// Load and run each plugin
			for _, pluginMeta := range plugins {
				if err := runner.LoadAndRunPlugin(ctx, pluginMeta.ID); err != nil {
					runner.logger.Error("Failed to auto-load plugin",
						"plugin_id", pluginMeta.ID,
						"plugin_name", pluginMeta.Name,
						"error", err.Error())
					continue
				}

				runner.logger.Info("Auto-loaded plugin",
					"plugin_id", pluginMeta.ID,
					"plugin_name", pluginMeta.Name)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			runner.StopAll()
			return nil
		},
	})
}
