package main

import (
	"fmt"
	"time"

	kronosTypes "github.com/backtesting-org/kronos-sdk/pkg/types/kronos"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MomentumStrategy implements momentum trading using Kronos SDK
type MomentumStrategy struct {
	*strategy.BaseStrategy
	k      kronosTypes.Kronos
	config MomentumConfig
}

// SetKronos injects the Kronos context at runtime
func (ms *MomentumStrategy) SetKronos(k kronosTypes.Kronos) { ms.k = k }

// MomentumConfig holds momentum strategy parameters
type MomentumConfig struct {
	BuyThreshold  decimal.Decimal // % change to trigger buy
	SellThreshold decimal.Decimal // % change to trigger sell
	OrderQuantity decimal.Decimal // Base quantity per order
	KlineInterval string          // Kline interval (e.g., "5m")
	KlineLimit    int             // Number of klines to analyze
}

// NewMomentumStrategy creates a new momentum strategy instance
func NewMomentumStrategy(k kronosTypes.Kronos, config MomentumConfig) *MomentumStrategy {
	base := strategy.NewBaseStrategy(
		strategy.Momentum,
		"Kline-based price momentum strategy",
		strategy.RiskLevelMedium,
		strategy.StrategyTypeMomentum,
	)

	return &MomentumStrategy{
		BaseStrategy: base,
		k:            k,
		config:       config,
	}
}

// GetSignals generates trading signals based on price momentum
func (ms *MomentumStrategy) GetSignals() ([]*strategy.Signal, error) {
	if !ms.IsEnabled() {
		return nil, nil
	}

    if ms.k == nil {
        return nil, nil
    }

    ms.k.Log().Info("Momentum", "", "Scanning for momentum opportunities...")

	var signals []*strategy.Signal

	// Assets to monitor
	assets := []string{"BTC", "ETH"}
	exchanges := []connector.ExchangeName{connector.Bybit, connector.Paradex}

	for _, assetSymbol := range assets {
		for _, exchange := range exchanges {
			signal := ms.generateMomentumSignal(assetSymbol, exchange)
			if signal != nil {
				signals = append(signals, signal)
			}
		}
	}

    if ms.k != nil {
        ms.k.Log().Info("Momentum", "", fmt.Sprintf("Generated %d momentum signals", len(signals)))
    }
	return signals, nil
}

// generateMomentumSignal generates a signal for a specific asset/exchange pair
func (ms *MomentumStrategy) generateMomentumSignal(
	assetSymbol string,
	exchange connector.ExchangeName,
) *strategy.Signal {
    if ms.k == nil {
        return nil
    }
    asset := ms.k.Asset(assetSymbol)

	// Get recent klines using Kronos store
	klines := ms.k.Store().GetKlines(asset, exchange, ms.config.KlineInterval, ms.config.KlineLimit)

	if len(klines) < 3 {
		return nil
	}

	current := klines[len(klines)-1]
	previous := klines[len(klines)-2]

	// Calculate price change percentage
	priceChange := current.Close.Sub(previous.Close).Div(previous.Close).Mul(decimal.NewFromInt(100))

    if ms.k != nil {
        ms.k.Log().Info("Momentum", assetSymbol, fmt.Sprintf("Price change: %s%% (Buy: >%s%%, Sell: <%s%%)",
			priceChange.StringFixed(4),
			ms.config.BuyThreshold.StringFixed(2),
			ms.config.SellThreshold.StringFixed(2)))
    }

	// Generate buy signal on positive momentum
	if priceChange.GreaterThan(ms.config.BuyThreshold) {
        if ms.k != nil {
            ms.k.Log().Opportunity("Momentum", assetSymbol, fmt.Sprintf("BUY signal on %s: %s%% change (threshold: %s%%)",
			exchange,
			priceChange.StringFixed(2),
            ms.config.BuyThreshold.StringFixed(2)))
        }
		return ms.createSignal(strategy.ActionBuy, assetSymbol, exchange, ms.config.OrderQuantity, current.Close)
	}

	// Generate sell signal on negative momentum
    if priceChange.LessThan(ms.config.SellThreshold) {
        if ms.k != nil {
            ms.k.Log().Opportunity("Momentum", assetSymbol, fmt.Sprintf("SELL signal on %s: %s%% change (threshold: %s%%)",
			exchange,
			priceChange.StringFixed(2),
            ms.config.SellThreshold.StringFixed(2)))
        }
		return ms.createSignal(strategy.ActionSell, assetSymbol, exchange, ms.config.OrderQuantity, current.Close)
	}

	return nil
}

// createSignal creates a trading signal
func (ms *MomentumStrategy) createSignal(
	action strategy.Action,
	assetSymbol string,
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
				Asset:    ms.k.Asset(assetSymbol),
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
	// Create with nil Kronos and default config for metadata extraction
	return NewMomentumStrategy(
		nil, // Kronos not needed for metadata
		MomentumConfig{
			BuyThreshold:  decimal.NewFromFloat(0.04),
			SellThreshold: decimal.NewFromFloat(-0.04),
			OrderQuantity: decimal.NewFromFloat(0.001),
			KlineInterval: "5m",
			KlineLimit:    20,
		},
	)
}

// Plugin export - required for Go plugin system
var Strategy strategy.Strategy = NewStrategy()
