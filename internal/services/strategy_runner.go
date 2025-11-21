package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/kronos/trade"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/google/uuid"
)

// StrategyRunner manages the execution of trading strategies
type StrategyRunner struct {
	pluginManager    plugin.Manager
	strategyRegistry registry.StrategyRegistry
	tradeService     *trade.TradeService
	logger           logging.ApplicationLogger

	runningStrategies map[uuid.UUID]*RunningStrategy
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// RunningStrategy tracks a running strategy instance
type RunningStrategy struct {
	ID           uuid.UUID
	PluginID     uuid.UUID
	Strategy     strategy.Strategy
	Ticker       *time.Ticker
	StopChan     chan struct{}
}

// NewStrategyRunner creates a new strategy runner
func NewStrategyRunner(
	pluginManager plugin.Manager,
	strategyRegistry registry.StrategyRegistry,
	tradeService *trade.TradeService,
	logger logging.ApplicationLogger,
) *StrategyRunner {
	ctx, cancel := context.WithCancel(context.Background())

	return &StrategyRunner{
		pluginManager:     pluginManager,
		strategyRegistry:  strategyRegistry,
		tradeService:      tradeService,
		logger:            logger,
		runningStrategies: make(map[uuid.UUID]*RunningStrategy),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// LoadAndRunPlugin loads a plugin and starts running its strategy
func (sr *StrategyRunner) LoadAndRunPlugin(ctx context.Context, pluginID uuid.UUID) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Check if already running
	if _, exists := sr.runningStrategies[pluginID]; exists {
		return fmt.Errorf("plugin %s is already running", pluginID)
	}

	// Instantiate strategy from plugin
	strat, err := sr.pluginManager.InstantiateStrategy(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("failed to instantiate strategy: %w", err)
	}

	// Register strategy
	sr.strategyRegistry.RegisterStrategy(strat)

	// Enable strategy
	if err := sr.strategyRegistry.EnableStrategy(strat.GetName()); err != nil {
		return fmt.Errorf("failed to enable strategy: %w", err)
	}

	// Create running strategy
	runID := uuid.New()
	ticker := time.NewTicker(10 * time.Second) // Run every 10 seconds
	stopChan := make(chan struct{})

	running := &RunningStrategy{
		ID:       runID,
		PluginID: pluginID,
		Strategy: strat,
		Ticker:   ticker,
		StopChan: stopChan,
	}

	sr.runningStrategies[pluginID] = running

	// Start execution loop
	go sr.executeStrategyLoop(running)

	sr.logger.Info("Strategy started",
		"plugin_id", pluginID,
		"strategy_name", strat.GetName(),
		"run_id", runID)

	return nil
}

// executeStrategyLoop runs the strategy execution loop
func (sr *StrategyRunner) executeStrategyLoop(running *RunningStrategy) {
	for {
		select {
		case <-running.Ticker.C:
			sr.executeStrategy(running)
		case <-running.StopChan:
			running.Ticker.Stop()
			return
		case <-sr.ctx.Done():
			running.Ticker.Stop()
			return
		}
	}
}

// executeStrategy executes a single iteration of the strategy
func (sr *StrategyRunner) executeStrategy(running *RunningStrategy) {
	defer func() {
		if r := recover(); r != nil {
			sr.logger.Error("Strategy execution panicked",
				"strategy", running.Strategy.GetName(),
				"error", r)
		}
	}()

	// Get signals from strategy (no parameters in SDK design)
	signals, err := running.Strategy.GetSignals()
	if err != nil {
		sr.logger.Error("Strategy failed to generate signals",
			"strategy", running.Strategy.GetName(),
			"error", err.Error())
		return
	}

	if len(signals) == 0 {
		return
	}

	sr.logger.Info("Strategy generated signals",
		"strategy", running.Strategy.GetName(),
		"signal_count", len(signals))

	// Execute trades for each signal using TradeService
	for _, signal := range signals {
		sr.logger.Info("Processing signal",
			"strategy", running.Strategy.GetName(),
			"signal_id", signal.ID,
			"action_count", len(signal.Actions))

		if err := sr.executeSignal(signal); err != nil {
			sr.logger.Error("Failed to execute signal",
				"strategy", running.Strategy.GetName(),
				"signal_id", signal.ID,
				"error", err.Error())
		}
	}
}

// executeSignal executes a single trading signal
func (sr *StrategyRunner) executeSignal(signal *strategy.Signal) error {
	// Iterate through all actions in the signal
	for _, action := range signal.Actions {
		if err := sr.executeAction(action); err != nil {
			sr.logger.Error("Failed to execute action",
				"strategy", signal.Strategy,
				"action", action.Action,
				"asset", action.Asset.Symbol(),
				"error", err.Error())
			// Continue with other actions even if one fails
			continue
		}

		sr.logger.Info("Action executed successfully",
			"strategy", signal.Strategy,
			"action", action.Action,
			"asset", action.Asset.Symbol(),
			"quantity", action.Quantity)
	}

	return nil
}

// executeAction executes a single trade action
func (sr *StrategyRunner) executeAction(action strategy.TradeAction) error {
	switch action.Action {
	case strategy.ActionBuy:
		_, err := sr.tradeService.Buy(action.Asset, action.Exchange, action.Quantity)
		return err

	case strategy.ActionSell:
		_, err := sr.tradeService.Sell(action.Asset, action.Exchange, action.Quantity)
		return err

	case strategy.ActionSellShort:
		_, err := sr.tradeService.Short(action.Asset, action.Exchange, action.Quantity)
		return err

	case strategy.ActionCover:
		_, err := sr.tradeService.CloseShort(action.Asset, action.Exchange, action.Quantity)
		return err

	default:
		return fmt.Errorf("unknown action type: %s", action.Action)
	}
}

// StopStrategy stops a running strategy
func (sr *StrategyRunner) StopStrategy(pluginID uuid.UUID) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	running, exists := sr.runningStrategies[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s is not running", pluginID)
	}

	// Stop the execution loop
	close(running.StopChan)

	// Disable strategy
	if err := sr.strategyRegistry.DisableStrategy(running.Strategy.GetName()); err != nil {
		sr.logger.Warn("Failed to disable strategy", "error", err.Error())
	}

	// Remove from running strategies
	delete(sr.runningStrategies, pluginID)

	sr.logger.Info("Strategy stopped", "plugin_id", pluginID)

	return nil
}

// StopAll stops all running strategies
func (sr *StrategyRunner) StopAll() {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	for pluginID := range sr.runningStrategies {
		running := sr.runningStrategies[pluginID]
		close(running.StopChan)
		sr.strategyRegistry.DisableStrategy(running.Strategy.GetName())
	}

	sr.runningStrategies = make(map[uuid.UUID]*RunningStrategy)
	sr.cancel()

	sr.logger.Info("All strategies stopped")
}

// GetRunningStrategies returns a list of currently running strategies
func (sr *StrategyRunner) GetRunningStrategies() []uuid.UUID {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(sr.runningStrategies))
	for id := range sr.runningStrategies {
		ids = append(ids, id)
	}

	return ids
}
