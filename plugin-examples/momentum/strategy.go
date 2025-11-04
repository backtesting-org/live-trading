package main

import (
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MomentumStrategy implements momentum trading using only Kronos SDK interfaces
type MomentumStrategy struct {
	*strategy.BaseStrategy
	assetStore store.Store
	logger     logging.ApplicationLogger
	config     MomentumConfig
}

// MomentumConfig holds momentum strategy parameters
type MomentumConfig struct {
	BuyThreshold  decimal.Decimal // % change to trigger buy
	SellThreshold decimal.Decimal // % change to trigger sell
	OrderQuantity decimal.Decimal // Base quantity per order
	KlineInterval string          // Kline interval (e.g., "5m")
	KlineLimit    int             // Number of klines to analyze
}

// NewMomentumStrategy creates a new momentum strategy instance
func NewMomentumStrategy(
	assetStore store.Store,
	logger logging.ApplicationLogger,
	config MomentumConfig,
) *MomentumStrategy {
	base := strategy.NewBaseStrategy(
		strategy.Momentum,
		"Kline-based price momentum strategy",
		strategy.RiskLevelMedium,
		strategy.StrategyTypeMomentum,
	)

	return &MomentumStrategy{
		BaseStrategy: base,
		assetStore:   assetStore,
		logger:       logger,
		config:       config,
	}
}

// GetSignals generates trading signals based on price momentum
func (ms *MomentumStrategy) GetSignals() ([]*strategy.Signal, error) {
	if !ms.IsEnabled() {
		return nil, nil
	}

	var signals []*strategy.Signal

	// Assets to monitor
	assets := []string{"BTC", "ETH"}
	exchanges := []connector.ExchangeName{connector.Bybit, connector.Paradex}

	for _, assetSymbol := range assets {
		asset := portfolio.NewAsset(assetSymbol)

		for _, exchange := range exchanges {
			signal := ms.generateMomentumSignal(asset, exchange)
			if signal != nil {
				signals = append(signals, signal)
			}
		}
	}

	ms.logger.Info("Momentum strategy generated %d signals", len(signals))
	return signals, nil
}

// generateMomentumSignal generates a signal for a specific asset/exchange pair
func (ms *MomentumStrategy) generateMomentumSignal(
	asset portfolio.Asset,
	exchange connector.ExchangeName,
) *strategy.Signal {
	// Get recent klines
	klines := ms.assetStore.GetKlines(asset, exchange, ms.config.KlineInterval, ms.config.KlineLimit)

	if len(klines) < 3 {
		return nil
	}

	current := klines[len(klines)-1]
	previous := klines[len(klines)-2]

	// Calculate price change percentage
	priceChange := current.Close.Sub(previous.Close).Div(previous.Close).Mul(decimal.NewFromInt(100))

	ms.logger.Info("Price change for %s on %s: %s%%", asset.Symbol(), exchange, priceChange.StringFixed(2))

	// Generate buy signal on positive momentum
	if priceChange.GreaterThan(ms.config.BuyThreshold) {
		ms.logger.Info("BUY signal for %s: %s%% change", asset.Symbol(), priceChange.StringFixed(2))
		return ms.createSignal(strategy.ActionBuy, asset, exchange, ms.config.OrderQuantity, current.Close)
	}

	// Generate sell signal on negative momentum
	if priceChange.LessThan(ms.config.SellThreshold) {
		ms.logger.Info("SELL signal for %s: %s%% change", asset.Symbol(), priceChange.StringFixed(2))
		return ms.createSignal(strategy.ActionSell, asset, exchange, ms.config.OrderQuantity, current.Close)
	}

	return nil
}

// createSignal creates a trading signal
func (ms *MomentumStrategy) createSignal(
	action strategy.Action,
	asset portfolio.Asset,
	exchange connector.ExchangeName,
	quantity decimal.Decimal,
	price decimal.Decimal,
) *strategy.Signal {
	return &strategy.Signal{
		ID:       uuid.New(),
		Strategy: ms.GetName(),
		Actions: []strategy.TradeAction{
			{
				Action:   action,
				Asset:    asset,
				Exchange: exchange,
				Quantity: quantity,
				Price:    price,
			},
		},
		Timestamp: time.Now(),
	}
}

// NewStrategy creates a new strategy instance for plugin loading
// This is called by the plugin manager to extract metadata
func NewStrategy() strategy.Strategy {
	// Create with default config for metadata extraction
	return NewMomentumStrategy(
		nil, // assetStore not needed for metadata
		nil, // logger not needed for metadata
		MomentumConfig{
			BuyThreshold:  decimal.NewFromFloat(2.0),
			SellThreshold: decimal.NewFromFloat(-2.0),
			OrderQuantity: decimal.NewFromFloat(0.01),
			KlineInterval: "5m",
			KlineLimit:    20,
		},
	)
}

// Plugin export - required for Go plugin system
var Strategy strategy.Strategy = NewStrategy()
