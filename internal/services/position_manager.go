package services

import (
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Position represents an open trading position
type Position struct {
	Asset      string
	Exchange   string
	Quantity   decimal.Decimal
	EntryPrice decimal.Decimal
	OrderID    string
	RunID      string          // Strategy run ID that opened this position
}

// TradePerformance tracks P&L metrics per strategy run
type TradePerformance struct {
	RunID           string
	RealizedPnL     decimal.Decimal // Total realized profit/loss
	UnrealizedPnL   decimal.Decimal // Current unrealized P&L
	WinningTrades   int             // Number of profitable trades
	LosingTrades    int             // Number of losing trades
	TotalVolume     decimal.Decimal // Total trading volume
	LargestWin      decimal.Decimal // Largest winning trade
	LargestLoss     decimal.Decimal // Largest losing trade
}

// PositionManager manages open positions and enforces risk limits
type PositionManager struct {
	mu        sync.RWMutex
	positions map[string]*Position            // key: asset+exchange
	performance map[string]*TradePerformance  // key: run_id
	logger    *zap.Logger

	// Risk limits
	maxPositionSize      decimal.Decimal
	maxTotalExposure     decimal.Decimal
	maxConcurrentTrades  int
}

// NewPositionManager creates a new position manager
func NewPositionManager(logger *zap.Logger) *PositionManager {
	return &PositionManager{
		positions:           make(map[string]*Position),
		performance:         make(map[string]*TradePerformance),
		logger:              logger,
		maxPositionSize:     decimal.NewFromFloat(10000), // $10k max per position
		maxTotalExposure:    decimal.NewFromFloat(50000), // $50k total exposure
		maxConcurrentTrades: 10,                          // max 10 concurrent trades
	}
}

// CheckRiskLimits validates if a trade action is within risk parameters
func (pm *PositionManager) CheckRiskLimits(action strategy.TradeAction) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Calculate position value
	positionValue := action.Quantity.Mul(action.Price)

	// Check single position size limit
	if positionValue.GreaterThan(pm.maxPositionSize) {
		return fmt.Errorf("position size %s exceeds limit %s",
			positionValue.String(),
			pm.maxPositionSize.String())
	}

	// Check total exposure limit
	totalExposure := pm.calculateTotalExposure()
	if totalExposure.Add(positionValue).GreaterThan(pm.maxTotalExposure) {
		return fmt.Errorf("total exposure %s would exceed limit %s",
			totalExposure.Add(positionValue).String(),
			pm.maxTotalExposure.String())
	}

	// Check concurrent trades limit
	if len(pm.positions) >= pm.maxConcurrentTrades {
		return fmt.Errorf("max concurrent trades (%d) reached", pm.maxConcurrentTrades)
	}

	return nil
}

// UpdatePosition updates position state after trade execution
func (pm *PositionManager) UpdatePosition(action strategy.TradeAction, orderID string, runID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := pm.getPositionKey(action.Asset.Symbol(), string(action.Exchange))

	switch action.Action {
	case strategy.ActionBuy:
		// Add or update position
		if existing, exists := pm.positions[key]; exists {
			// Average entry price
			totalValue := existing.Quantity.Mul(existing.EntryPrice).Add(action.Quantity.Mul(action.Price))
			totalQuantity := existing.Quantity.Add(action.Quantity)
			existing.Quantity = totalQuantity
			existing.EntryPrice = totalValue.Div(totalQuantity)
		} else {
			pm.positions[key] = &Position{
				Asset:      action.Asset.Symbol(),
				Exchange:   string(action.Exchange),
				Quantity:   action.Quantity,
				EntryPrice: action.Price,
				OrderID:    orderID,
				RunID:      runID,
			}
		}

	case strategy.ActionSell:
		// Reduce or close position
		if existing, exists := pm.positions[key]; exists {
			existing.Quantity = existing.Quantity.Sub(action.Quantity)
			if existing.Quantity.LessThanOrEqual(decimal.Zero) {
				delete(pm.positions, key)
				pm.logger.Info("Position closed",
					zap.String("asset", action.Asset.Symbol()),
					zap.String("exchange", string(action.Exchange)))
			}
		}
	}

	pm.logger.Debug("Position updated",
		zap.String("key", key),
		zap.String("action", string(action.Action)),
		zap.Int("total_positions", len(pm.positions)))
}

// GetPosition retrieves current position for an asset
func (pm *PositionManager) GetPosition(asset, exchange string) *Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.getPositionKey(asset, exchange)
	return pm.positions[key]
}

// GetAllPositions returns all open positions
func (pm *PositionManager) GetAllPositions() map[string]*Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy to avoid race conditions
	positions := make(map[string]*Position)
	for k, v := range pm.positions {
		positions[k] = &Position{
			Asset:      v.Asset,
			Exchange:   v.Exchange,
			Quantity:   v.Quantity,
			EntryPrice: v.EntryPrice,
			OrderID:    v.OrderID,
			RunID:      v.RunID,
		}
	}

	return positions
}

// GetPositionsByRunID returns all positions for a specific strategy run
func (pm *PositionManager) GetPositionsByRunID(runID string) []*Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	positions := make([]*Position, 0)
	for _, v := range pm.positions {
		if v.RunID == runID {
			positions = append(positions, &Position{
				Asset:      v.Asset,
				Exchange:   v.Exchange,
				Quantity:   v.Quantity,
				EntryPrice: v.EntryPrice,
				OrderID:    v.OrderID,
				RunID:      v.RunID,
			})
		}
	}

	return positions
}

// CalculatePnL calculates unrealized PnL for a position
func (pm *PositionManager) CalculatePnL(asset, exchange string, currentPrice decimal.Decimal) decimal.Decimal {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	key := pm.getPositionKey(asset, exchange)
	position, exists := pm.positions[key]
	if !exists {
		return decimal.Zero
	}

	priceDiff := currentPrice.Sub(position.EntryPrice)
	pnl := priceDiff.Mul(position.Quantity)

	return pnl
}

// calculateTotalExposure calculates total notional exposure across all positions
func (pm *PositionManager) calculateTotalExposure() decimal.Decimal {
	total := decimal.Zero

	for _, position := range pm.positions {
		exposure := position.Quantity.Mul(position.EntryPrice)
		total = total.Add(exposure)
	}

	return total
}

// getPositionKey creates a unique key for position tracking
func (pm *PositionManager) getPositionKey(asset, exchange string) string {
	return fmt.Sprintf("%s_%s", asset, exchange)
}

// SetRiskLimits updates risk management parameters
func (pm *PositionManager) SetRiskLimits(
	maxPositionSize,
	maxTotalExposure decimal.Decimal,
	maxConcurrentTrades int,
) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.maxPositionSize = maxPositionSize
	pm.maxTotalExposure = maxTotalExposure
	pm.maxConcurrentTrades = maxConcurrentTrades

	pm.logger.Info("Risk limits updated",
		zap.String("max_position", maxPositionSize.String()),
		zap.String("max_exposure", maxTotalExposure.String()),
		zap.Int("max_trades", maxConcurrentTrades))
}

// RecordTrade records a completed trade and updates P&L metrics
func (pm *PositionManager) RecordTrade(runID string, action strategy.TradeAction, exitPrice decimal.Decimal) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Initialize performance tracking for this run if needed
	if pm.performance[runID] == nil {
		pm.performance[runID] = &TradePerformance{
			RunID:           runID,
			RealizedPnL:     decimal.Zero,
			UnrealizedPnL:   decimal.Zero,
			WinningTrades:   0,
			LosingTrades:    0,
			TotalVolume:     decimal.Zero,
			LargestWin:      decimal.Zero,
			LargestLoss:     decimal.Zero,
		}
	}

	perf := pm.performance[runID]

	// Calculate P&L for this trade
	var pnl decimal.Decimal
	if action.Action == strategy.ActionSell {
		// For sells, we're closing a position - calculate realized P&L
		pnl = exitPrice.Sub(action.Price).Mul(action.Quantity)

		// Update realized P&L
		perf.RealizedPnL = perf.RealizedPnL.Add(pnl)

		// Track win/loss
		if pnl.GreaterThan(decimal.Zero) {
			perf.WinningTrades++
			if pnl.GreaterThan(perf.LargestWin) {
				perf.LargestWin = pnl
			}
		} else if pnl.LessThan(decimal.Zero) {
			perf.LosingTrades++
			if pnl.LessThan(perf.LargestLoss) {
				perf.LargestLoss = pnl
			}
		}
	}

	// Update total volume
	tradeVolume := action.Quantity.Mul(action.Price)
	perf.TotalVolume = perf.TotalVolume.Add(tradeVolume)

	pm.logger.Debug("Trade recorded",
		zap.String("run_id", runID),
		zap.String("action", string(action.Action)),
		zap.String("pnl", pnl.String()),
		zap.String("total_pnl", perf.RealizedPnL.String()))
}

// GetPerformance retrieves performance metrics for a strategy run
func (pm *PositionManager) GetPerformance(runID string) *TradePerformance {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if perf, exists := pm.performance[runID]; exists {
		// Return a copy
		return &TradePerformance{
			RunID:           perf.RunID,
			RealizedPnL:     perf.RealizedPnL,
			UnrealizedPnL:   perf.UnrealizedPnL,
			WinningTrades:   perf.WinningTrades,
			LosingTrades:    perf.LosingTrades,
			TotalVolume:     perf.TotalVolume,
			LargestWin:      perf.LargestWin,
			LargestLoss:     perf.LargestLoss,
		}
	}

	// Return empty performance if not found
	return &TradePerformance{
		RunID:           runID,
		RealizedPnL:     decimal.Zero,
		UnrealizedPnL:   decimal.Zero,
		WinningTrades:   0,
		LosingTrades:    0,
		TotalVolume:     decimal.Zero,
		LargestWin:      decimal.Zero,
		LargestLoss:     decimal.Zero,
	}
}

// GetWinRate calculates the win rate percentage for a strategy run
func (pm *PositionManager) GetWinRate(runID string) decimal.Decimal {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	perf, exists := pm.performance[runID]
	if !exists {
		return decimal.Zero
	}

	totalTrades := perf.WinningTrades + perf.LosingTrades
	if totalTrades == 0 {
		return decimal.Zero
	}

	winRate := decimal.NewFromInt(int64(perf.WinningTrades)).
		Div(decimal.NewFromInt(int64(totalTrades))).
		Mul(decimal.NewFromInt(100))

	return winRate
}

// ClearPerformance clears performance data for a strategy run (when it stops)
func (pm *PositionManager) ClearPerformance(runID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.performance, runID)
	pm.logger.Debug("Performance data cleared", zap.String("run_id", runID))
}
