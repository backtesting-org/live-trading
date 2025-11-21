package exchange

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides exchange configurations
var Module = fx.Module("exchange-configs",
	fx.Provide(
		NewParadexConfig,
		// Add more exchange configs here as needed
	),
)

// NewParadexConfig loads Paradex configuration
func NewParadexConfig(logger *zap.Logger) (*Paradex, error) {
	logger.Info("Loading Paradex configuration...")
	cfg := &Paradex{}

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
