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

// GridStrategy implements grid trading using only Kronos SDK interfaces
type GridStrategy struct {
	*strategy.BaseStrategy
	assetStore store.Store
	logger     logging.ApplicationLogger
	config     GridConfig
}

// GridConfig holds grid trading parameters
type GridConfig struct {
	PriceLower          decimal.Decimal
	PriceUpper          decimal.Decimal
	GridStep            decimal.Decimal
	OrderSizeQuote      decimal.Decimal
	MaxConcurrentOrders int
}

// NewGridStrategy creates a new grid strategy instance
func NewGridStrategy(
	assetStore store.Store,
	logger logging.ApplicationLogger,
	config GridConfig,
) *GridStrategy {
	base := strategy.NewBaseStrategy(
		strategy.StrategyName("Grid Trading"),
		"Market-neutral grid trading strategy",
		strategy.RiskLevelMedium,
		strategy.StrategyTypeTechnical,
	)

	return &GridStrategy{
		BaseStrategy: base,
		assetStore:   assetStore,
		logger:       logger,
		config:       config,
	}
}

// GetSignals generates trading signals for grid strategy
func (gs *GridStrategy) GetSignals() ([]*strategy.Signal, error) {
	if !gs.IsEnabled() {
		return nil, nil
	}

	// Get BTC price
	asset := portfolio.NewAsset("BTC")
	exchange := connector.Bybit

	// Get order book
	orderBook := gs.assetStore.GetOrderBook(asset, exchange, connector.TypePerpetual)
	if orderBook == nil || len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0 {
		gs.logger.Info("No orderbook data available for %s on %s", asset.Symbol(), exchange)
		return nil, nil
	}

	// Calculate current mid price
	currentPrice := orderBook.Bids[0].Price.Add(orderBook.Asks[0].Price).Div(decimal.NewFromInt(2))

	if currentPrice.LessThan(gs.config.PriceLower) || currentPrice.GreaterThan(gs.config.PriceUpper) {
		gs.logger.Info("Price %s outside grid range [%s-%s]",
			currentPrice.String(),
			gs.config.PriceLower.String(),
			gs.config.PriceUpper.String())
		return nil, nil
	}

	// Calculate grid levels
	buyLevels := gs.calculateBuyLevels(currentPrice)
	sellLevels := gs.calculateSellLevels(currentPrice)

	var signals []*strategy.Signal
	signalsGenerated := 0
	maxBuySignals := gs.config.MaxConcurrentOrders / 2

	// Generate buy signals
	for i, level := range buyLevels {
		if signalsGenerated >= gs.config.MaxConcurrentOrders {
			break
		}
		if i >= maxBuySignals {
			break
		}

		quantity := gs.calculateOrderQuantity(level)
		signal := gs.createBuySignal(asset, level, exchange, quantity)
		signals = append(signals, signal)
		signalsGenerated++
	}

	// Generate sell signals
	for _, level := range sellLevels {
		if signalsGenerated >= gs.config.MaxConcurrentOrders {
			break
		}

		quantity := gs.calculateOrderQuantity(level)
		signal := gs.createSellSignal(asset, level, exchange, quantity)
		signals = append(signals, signal)
		signalsGenerated++
	}

	gs.logger.Info("Generated %d grid signals for %s on %s", len(signals), asset.Symbol(), exchange)
	return signals, nil
}

// calculateBuyLevels calculates buy price levels below current price
func (gs *GridStrategy) calculateBuyLevels(currentPrice decimal.Decimal) []decimal.Decimal {
	var levels []decimal.Decimal
	level := currentPrice.Sub(gs.config.GridStep)

	for level.GreaterThanOrEqual(gs.config.PriceLower) {
		levels = append(levels, level)
		level = level.Sub(gs.config.GridStep)
	}

	return levels
}

// calculateSellLevels calculates sell price levels above current price
func (gs *GridStrategy) calculateSellLevels(currentPrice decimal.Decimal) []decimal.Decimal {
	var levels []decimal.Decimal
	level := currentPrice.Add(gs.config.GridStep)

	for level.LessThanOrEqual(gs.config.PriceUpper) {
		levels = append(levels, level)
		level = level.Add(gs.config.GridStep)
	}

	return levels
}

// calculateOrderQuantity calculates order quantity in base currency
func (gs *GridStrategy) calculateOrderQuantity(price decimal.Decimal) decimal.Decimal {
	if price.IsZero() {
		return decimal.Zero
	}
	return gs.config.OrderSizeQuote.Div(price)
}

// createBuySignal creates a buy signal
func (gs *GridStrategy) createBuySignal(
	asset portfolio.Asset,
	price decimal.Decimal,
	exchange connector.ExchangeName,
	quantity decimal.Decimal,
) *strategy.Signal {
	return &strategy.Signal{
		ID:       uuid.New(),
		Strategy: gs.GetName(),
		Actions: []strategy.TradeAction{
			{
				Action:   strategy.ActionBuy,
				Asset:    asset,
				Exchange: exchange,
				Quantity: quantity,
				Price:    price,
			},
		},
		Timestamp: time.Now(),
	}
}

// createSellSignal creates a sell signal
func (gs *GridStrategy) createSellSignal(
	asset portfolio.Asset,
	price decimal.Decimal,
	exchange connector.ExchangeName,
	quantity decimal.Decimal,
) *strategy.Signal {
	return &strategy.Signal{
		ID:       uuid.New(),
		Strategy: gs.GetName(),
		Actions: []strategy.TradeAction{
			{
				Action:   strategy.ActionSell,
				Asset:    asset,
				Exchange: exchange,
				Quantity: quantity,
				Price:    price,
			},
		},
		Timestamp: time.Now(),
	}
}

// Plugin export - required for Go plugin system
var Strategy strategy.Strategy
