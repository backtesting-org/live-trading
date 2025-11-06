package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StrategyExecutor manages strategy execution lifecycle
type StrategyExecutor struct {
	repo            *database.Repository
	pluginManager   *PluginManager
	kronosProvider  *KronosProvider
	tradeExecutor   *TradeExecutor
	marketDataFeed  *MarketDataFeed
	logger          *zap.Logger
	runningStrategies map[uuid.UUID]*RunningStrategy
	mu              sync.RWMutex
	eventBus        *EventBus
}

// RunningStrategy represents an actively running strategy
type RunningStrategy struct {
	RunID        uuid.UUID
	PluginID     uuid.UUID
	Strategy     strategy.Strategy
	CancelFunc   context.CancelFunc
	StartTime    time.Time
	Stats        *RuntimeStats
	ErrorChan    chan error
}

// RuntimeStats tracks runtime metrics for a strategy
type RuntimeStats struct {
	mu           sync.RWMutex
	TotalSignals int64
	TotalTrades  int64
	ErrorCount   int64
	LastSignal   time.Time
	CPUUsage     float64
	MemoryUsage  int64
}

// NewStrategyExecutor creates a new strategy executor
func NewStrategyExecutor(
	repo *database.Repository,
	pluginManager *PluginManager,
	kronosProvider *KronosProvider,
	tradeExecutor *TradeExecutor,
	marketDataFeed *MarketDataFeed,
	logger *zap.Logger,
	eventBus *EventBus,
) *StrategyExecutor {
	return &StrategyExecutor{
		repo:              repo,
		pluginManager:     pluginManager,
		kronosProvider:    kronosProvider,
		tradeExecutor:     tradeExecutor,
		marketDataFeed:    marketDataFeed,
		logger:            logger,
		runningStrategies: make(map[uuid.UUID]*RunningStrategy),
		eventBus:          eventBus,
	}
}

// StartStrategy starts a strategy execution
func (se *StrategyExecutor) StartStrategy(ctx context.Context, pluginID uuid.UUID, configID *uuid.UUID) (uuid.UUID, error) {
	se.mu.Lock()
	defer se.mu.Unlock()

	// Check if strategy is already running
	activeRun, err := se.repo.GetActiveRun(ctx, pluginID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to check active run: %w", err)
	}
	if activeRun != nil {
		return uuid.Nil, fmt.Errorf("strategy is already running with run ID: %s", activeRun.ID)
	}

    // Instantiate strategy from plugin
    strat, err := se.pluginManager.InstantiateStrategy(ctx, pluginID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to instantiate strategy: %w", err)
	}

    // Inject Kronos if strategy supports it
    if se.kronosProvider != nil {
        if aware, ok := strat.(KronosAware); ok {
            k := se.kronosProvider.CreateKronos()
            aware.SetKronos(k)

            // Register core assets for market data feed (Paradex-only support)
            if se.marketDataFeed != nil {
                se.marketDataFeed.AddAsset(k.Asset("BTC"))
            }
        }
    }

	// Create run record
	run := &database.StrategyRun{
		ID:        uuid.New(),
		PluginID:  pluginID,
		ConfigID:  configID,
		Status:    database.RunStatusRunning,
		StartTime: time.Now(),
	}

	if err := se.repo.CreateRun(ctx, run); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create run record: %w", err)
	}

	// Enable the strategy
	if err := strat.Enable(); err != nil {
		// Update run status to error
		run.Status = database.RunStatusError
		errorMsg := err.Error()
		run.ErrorMessage = &errorMsg
		se.repo.UpdateRun(ctx, run)
		return uuid.Nil, fmt.Errorf("failed to enable strategy: %w", err)
	}

	// Create context with cancellation
	runCtx, cancel := context.WithCancel(context.Background())

	// Track running strategy
	running := &RunningStrategy{
		RunID:      run.ID,
		PluginID:   pluginID,
		Strategy:   strat,
		CancelFunc: cancel,
		StartTime:  time.Now(),
		Stats:      &RuntimeStats{},
		ErrorChan:  make(chan error, 10),
	}

	se.runningStrategies[run.ID] = running

	// Start execution loop in goroutine
	go se.executionLoop(runCtx, running)

	// Publish event
	se.eventBus.Publish(Event{
		Type: EventStrategyStarted,
		Data: map[string]interface{}{
			"run_id":    run.ID.String(),
			"plugin_id": pluginID.String(),
			"timestamp": time.Now(),
		},
	})

	se.logger.Info("Strategy started",
		zap.String("run_id", run.ID.String()),
		zap.String("plugin_id", pluginID.String()))

	return run.ID, nil
}

// StopStrategy stops a running strategy
func (se *StrategyExecutor) StopStrategy(ctx context.Context, runID uuid.UUID) error {
	se.mu.Lock()
	running, exists := se.runningStrategies[runID]
	if !exists {
		se.mu.Unlock()
		return fmt.Errorf("strategy run not found or not running")
	}
	delete(se.runningStrategies, runID)
	se.mu.Unlock()

	// Cancel the execution context
	running.CancelFunc()

	// Disable the strategy
	if err := running.Strategy.Disable(); err != nil {
		se.logger.Error("Error disabling strategy", zap.Error(err))
	}

	// Update run record
	run, err := se.repo.GetRun(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}

	endTime := time.Now()
	run.Status = database.RunStatusStopped
	run.EndTime = &endTime
	run.TotalSignals = running.Stats.TotalSignals
	run.TotalTrades = running.Stats.TotalTrades
	run.ErrorCount = running.Stats.ErrorCount
	run.CPUUsage = running.Stats.CPUUsage
	run.MemoryUsage = running.Stats.MemoryUsage

	if err := se.repo.UpdateRun(ctx, run); err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}

	// Publish event
	se.eventBus.Publish(Event{
		Type: EventStrategyStopped,
		Data: map[string]interface{}{
			"run_id":    runID.String(),
			"timestamp": time.Now(),
			"stats": map[string]interface{}{
				"total_signals": running.Stats.TotalSignals,
				"total_trades":  running.Stats.TotalTrades,
				"error_count":   running.Stats.ErrorCount,
			},
		},
	})

	se.logger.Info("Strategy stopped",
		zap.String("run_id", runID.String()),
		zap.Int64("total_signals", running.Stats.TotalSignals))

	return nil
}

// executionLoop runs the strategy signal generation loop
func (se *StrategyExecutor) executionLoop(ctx context.Context, running *RunningStrategy) {
	ticker := time.NewTicker(5 * time.Second) // Generate signals every 5 seconds
	defer ticker.Stop()

	defer func() {
		if r := recover(); r != nil {
			se.logger.Error("Strategy execution panic",
				zap.String("run_id", running.RunID.String()),
				zap.Any("panic", r))

			// Update run status
			updateCtx := context.Background()
			run, err := se.repo.GetRun(updateCtx, running.RunID)
			if err == nil {
				endTime := time.Now()
				run.Status = database.RunStatusError
				errorMsg := fmt.Sprintf("panic: %v", r)
				run.ErrorMessage = &errorMsg
				run.EndTime = &endTime
				se.repo.UpdateRun(updateCtx, run)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			se.logger.Info("Execution loop cancelled", zap.String("run_id", running.RunID.String()))
			return

		case <-ticker.C:
			// Get signals from strategy
			signals, err := running.Strategy.GetSignals()
			if err != nil {
				se.logger.Error("Error getting signals",
					zap.String("run_id", running.RunID.String()),
					zap.Error(err))

				running.Stats.mu.Lock()
				running.Stats.ErrorCount++
				running.Stats.mu.Unlock()

				// Log error to database
				se.logError(ctx, running.RunID, err)

				// Check if too many errors
				if running.Stats.ErrorCount > 10 {
					se.logger.Error("Too many errors, stopping strategy",
						zap.String("run_id", running.RunID.String()))

					// Stop the strategy
					go se.StopStrategy(context.Background(), running.RunID)
					return
				}

				continue
			}

			// Process signals
			if len(signals) > 0 {
				se.processSignals(ctx, running, signals)
			}

			// Update runtime metrics
			se.updateMetrics(ctx, running)
		}
	}
}

// processSignals processes trading signals and executes them
func (se *StrategyExecutor) processSignals(ctx context.Context, running *RunningStrategy, signals []*strategy.Signal) {
	for _, signal := range signals {
		// Execute the signal on the exchange
		if err := se.tradeExecutor.ExecuteSignal(ctx, running.RunID, signal); err != nil {
			se.logger.Error("Failed to execute signal",
				zap.String("run_id", running.RunID.String()),
				zap.String("signal_id", signal.ID.String()),
				zap.Error(err))

			running.Stats.mu.Lock()
			running.Stats.ErrorCount++
			running.Stats.mu.Unlock()

			continue
		}

		// Update stats
		running.Stats.mu.Lock()
		running.Stats.TotalSignals += 1
		running.Stats.TotalTrades += int64(len(signal.Actions))
		running.Stats.LastSignal = time.Now()
		running.Stats.mu.Unlock()

		se.logger.Info("Signal executed successfully",
			zap.String("run_id", running.RunID.String()),
			zap.String("signal_id", signal.ID.String()),
			zap.Int("actions", len(signal.Actions)))
	}
}

// updateMetrics updates runtime metrics
func (se *StrategyExecutor) updateMetrics(ctx context.Context, running *RunningStrategy) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	running.Stats.mu.Lock()
	running.Stats.MemoryUsage = int64(m.Alloc)
	// CPU monitoring needs platform-specific code, leaving at 0 for now
	running.Stats.CPUUsage = 0.0
	running.Stats.mu.Unlock()

	// Save to database every minute
	if time.Since(running.StartTime)%(1*time.Minute) < 5*time.Second {
		run, err := se.repo.GetRun(ctx, running.RunID)
		if err != nil {
			se.logger.Error("Failed to get run for metrics update", zap.Error(err))
			return
		}

		run.TotalSignals = running.Stats.TotalSignals
		run.TotalTrades = running.Stats.TotalTrades
		run.ErrorCount = running.Stats.ErrorCount
		run.CPUUsage = running.Stats.CPUUsage
		run.MemoryUsage = running.Stats.MemoryUsage

		if err := se.repo.UpdateRun(ctx, run); err != nil {
			se.logger.Error("Failed to update run metrics", zap.Error(err))
		}
	}
}

// logError logs an error to the database
func (se *StrategyExecutor) logError(ctx context.Context, runID uuid.UUID, err error) {
	log := &database.ExecutionLog{
		ID:        uuid.New(),
		RunID:     runID,
		Level:     database.LogLevelError,
		Message:   err.Error(),
		Metadata:  database.LogMetadata{},
		Timestamp: time.Now(),
	}

	if err := se.repo.CreateLog(ctx, log); err != nil {
		se.logger.Error("Failed to create error log", zap.Error(err))
	}
}

// GetRunStatus retrieves the current status of a running strategy
func (se *StrategyExecutor) GetRunStatus(ctx context.Context, runID uuid.UUID) (*ExecutionStatus, error) {
	// Get run from database
	run, err := se.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	status := &ExecutionStatus{
		RunID:      run.ID,
		PluginID:   run.PluginID,
		Status:     run.Status,
		StartTime:  run.StartTime,
		EndTime:    run.EndTime,
		TotalSignals: run.TotalSignals,
		TotalTrades:  run.TotalTrades,
		ErrorCount:   run.ErrorCount,
		CPUUsage:     run.CPUUsage,
		MemoryUsage:  run.MemoryUsage,
	}

	// If running, get live stats
	se.mu.RLock()
	running, exists := se.runningStrategies[runID]
	se.mu.RUnlock()

	if exists {
		running.Stats.mu.RLock()
		status.TotalSignals = running.Stats.TotalSignals
		status.TotalTrades = running.Stats.TotalTrades
		status.ErrorCount = running.Stats.ErrorCount
		status.LastSignal = &running.Stats.LastSignal
		status.CPUUsage = running.Stats.CPUUsage
		status.MemoryUsage = running.Stats.MemoryUsage
		running.Stats.mu.RUnlock()
	}

	return status, nil
}

// ListRuns retrieves execution history for a plugin
func (se *StrategyExecutor) ListRuns(ctx context.Context, pluginID uuid.UUID, limit, offset int) ([]database.StrategyRun, error) {
	return se.repo.ListRunsByPlugin(ctx, pluginID, limit, offset)
}

// GetRunStats retrieves detailed statistics for a run
func (se *StrategyExecutor) GetRunStats(ctx context.Context, runID uuid.UUID) (map[string]interface{}, error) {
	// Get base stats from database
	stats, err := se.repo.GetRunStats(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Add P&L and win rate from PositionManager
	perf := se.tradeExecutor.positionManager.GetPerformance(runID.String())
	winRate := se.tradeExecutor.positionManager.GetWinRate(runID.String())

	// Enhance stats with P&L metrics
	stats["realized_pnl"] = perf.RealizedPnL.String()
	stats["unrealized_pnl"] = perf.UnrealizedPnL.String()
	stats["winning_trades"] = perf.WinningTrades
	stats["losing_trades"] = perf.LosingTrades
	stats["win_rate"] = winRate.StringFixed(2) + "%" // e.g., "65.50%"
	stats["total_volume"] = perf.TotalVolume.String()
	stats["largest_win"] = perf.LargestWin.String()
	stats["largest_loss"] = perf.LargestLoss.String()

	return stats, nil
}

// ExecutionStatus represents the current status of a strategy execution
type ExecutionStatus struct {
	RunID        uuid.UUID   `json:"run_id"`
	PluginID     uuid.UUID   `json:"plugin_id"`
	Status       string      `json:"status"`
	StartTime    time.Time   `json:"start_time"`
	EndTime      *time.Time  `json:"end_time,omitempty"`
	TotalSignals int64       `json:"total_signals"`
	TotalTrades  int64       `json:"total_trades"`
	ErrorCount   int64       `json:"error_count"`
	LastSignal   *time.Time  `json:"last_signal,omitempty"`
	CPUUsage     float64     `json:"cpu_usage"`
	MemoryUsage  int64       `json:"memory_usage"`
}
