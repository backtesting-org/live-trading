package exchange

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides exchange configurations
var Module = fx.Module("exchange-configs",
	fx.Provide(
		NewParadexConfig,
		NewHyperliquidConfig,
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

// NewHyperliquidConfig loads Hyperliquid configuration
func NewHyperliquidConfig(logger *zap.Logger) (*HyperliquidConfig, error) {
	logger.Info("Loading Hyperliquid configuration...")
	cfg := &HyperliquidConfig{}

	// Try to load config, but don't fail if credentials are missing
	// This allows the server to start even without Hyperliquid credentials
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("Hyperliquid configuration incomplete - trading will be disabled until credentials are provided",
				zap.Any("error", r))
		}
	}()

	cfg.LoadHyperliquidConfig()
	return cfg, nil
}
