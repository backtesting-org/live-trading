package services

import (
	"github.com/backtesting-org/kronos-sdk/pkg/kronos"
	"github.com/backtesting-org/kronos-sdk/pkg/kronos/analytics"
	"github.com/backtesting-org/kronos-sdk/pkg/kronos/indicators"
	"github.com/backtesting-org/kronos-sdk/pkg/kronos/market"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store"
	"go.uber.org/zap"
)

// KronosProvider creates Kronos context instances for strategies
type KronosProvider struct {
	store         store.Store
	tradingLogger logging.TradingLogger
	logger        *zap.Logger
}

// NewKronosProvider creates a new Kronos provider
func NewKronosProvider(
	store store.Store,
	tradingLogger logging.TradingLogger,
	logger *zap.Logger,
) *KronosProvider {
	return &KronosProvider{
		store:         store,
		tradingLogger: tradingLogger,
		logger:        logger,
	}
}

// CreateKronos creates a new Kronos context with all services initialized
func (kp *KronosProvider) CreateKronos() *kronos.Kronos {
	// Create service instances
	indicatorService := indicators.NewIndicatorService(kp.store)
	marketService := market.NewMarketService(kp.store)
	analyticsService := analytics.NewAnalyticsService(kp.store)

	// Create Kronos context
	k := kronos.NewKronos(
		kp.store,
		kp.tradingLogger,
		indicatorService,
		marketService,
		analyticsService,
	)

	kp.logger.Debug("Created new Kronos context")
	return k
}
