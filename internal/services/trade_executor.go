package services

import (
	"context"
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/exchanges/paradex"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TradeExecutor executes trading signals on exchanges
type TradeExecutor struct {
	paradexConnector *paradex.Paradex
	positionManager  *PositionManager
	repo             *database.Repository
	eventBus         *EventBus
	logger           *zap.Logger
}

// NewTradeExecutor creates a new trade executor service
func NewTradeExecutor(
	paradexConnector *paradex.Paradex,
	positionManager *PositionManager,
	repo *database.Repository,
	eventBus *EventBus,
	logger *zap.Logger,
) *TradeExecutor {
	return &TradeExecutor{
		paradexConnector: paradexConnector,
		positionManager:  positionManager,
		repo:             repo,
		eventBus:         eventBus,
		logger:           logger,
	}
}

// ExecuteSignal processes a trading signal and executes it on the exchange
func (te *TradeExecutor) ExecuteSignal(ctx context.Context, runID uuid.UUID, signal *strategy.Signal) error {
	// Check if connector is available
	if te.paradexConnector == nil {
		return fmt.Errorf("paradex connector not available - cannot execute trades")
	}

	te.logger.Info("Executing signal",
		zap.String("run_id", runID.String()),
		zap.String("signal_id", signal.ID.String()),
		zap.String("strategy", string(signal.Strategy)))

	// Process each trade action in the signal
	for _, action := range signal.Actions {
		if err := te.executeTradeAction(ctx, runID, signal.ID, action); err != nil {
			te.logger.Error("Failed to execute trade action",
				zap.String("signal_id", signal.ID.String()),
				zap.String("asset", action.Asset.Symbol()),
				zap.Error(err))

			// Store error in execution logs
			te.logExecution(ctx, runID, "error", fmt.Sprintf("Failed to execute %s on %s: %v",
				action.Action, action.Asset.Symbol(), err))

			return err
		}
	}

	// Store signal in database
	if err := te.storeSignal(ctx, runID, signal); err != nil {
		te.logger.Error("Failed to store signal", zap.Error(err))
	}

	// Publish signal event
	te.eventBus.Publish(Event{
		Type: EventSignalGenerated,
		Data: map[string]interface{}{
			"run_id":    runID.String(),
			"signal_id": signal.ID.String(),
			"strategy":  signal.Strategy,
			"actions":   len(signal.Actions),
		},
	})

	return nil
}

// executeTradeAction executes a single trade action
func (te *TradeExecutor) executeTradeAction(
	ctx context.Context,
	runID uuid.UUID,
	signalID uuid.UUID,
	action strategy.TradeAction,
) error {
	// Check risk limits before executing
	if err := te.positionManager.CheckRiskLimits(action); err != nil {
		te.logger.Warn("Trade rejected by risk manager",
			zap.String("asset", action.Asset.Symbol()),
			zap.Error(err))
		return err
	}

	// Execute based on action type and exchange
	var orderID string
	var err error

	switch action.Exchange {
	case connector.Paradex:
		orderID, err = te.executeOnParadex(ctx, action)
	default:
		return fmt.Errorf("unsupported exchange: %s", action.Exchange)
	}

	if err != nil {
		return err
	}

	// Update position manager
	te.positionManager.UpdatePosition(action, orderID)

	// Record trade for P&L tracking (exit price is the action price for now)
	te.positionManager.RecordTrade(runID.String(), action, action.Price)

	// Log successful execution
	te.logExecution(ctx, runID, "info",
		fmt.Sprintf("Executed %s %s on %s, order: %s",
			action.Action,
			action.Asset.Symbol(),
			action.Exchange,
			orderID))

	te.logger.Info("Trade executed successfully",
		zap.String("order_id", orderID),
		zap.String("action", string(action.Action)),
		zap.String("asset", action.Asset.Symbol()),
		zap.String("quantity", action.Quantity.String()),
		zap.String("price", action.Price.String()))

	return nil
}

// executeOnParadex executes a trade on Paradex exchange
func (te *TradeExecutor) executeOnParadex(ctx context.Context, action strategy.TradeAction) (string, error) {
	var resp *connector.OrderResponse
	var err error

	side := connector.OrderSideBuy
	if action.Action == strategy.ActionSell {
		side = connector.OrderSideSell
	}

	// Use market order for simplicity - PlaceMarketOrder expects string symbol
	resp, err = te.paradexConnector.PlaceMarketOrder(action.Asset.Symbol(), side, action.Quantity)
	if err != nil {
		return "", err
	}

	return resp.OrderID, nil
}

// storeSignal saves a trading signal to the database
func (te *TradeExecutor) storeSignal(ctx context.Context, runID uuid.UUID, signal *strategy.Signal) error {
	for _, action := range signal.Actions {
		dbSignal := &database.TradingSignal{
			ID:         uuid.New(),
			RunID:      runID,
			SignalType: string(action.Action),
			Asset:      action.Asset.Symbol(),
			Exchange:   string(action.Exchange),
			Quantity:   action.Quantity,
			Price:      action.Price,
			Timestamp:  signal.Timestamp,
			Executed:   true, // Mark as executed
		}

		if err := te.repo.CreateSignal(ctx, dbSignal); err != nil {
			return err
		}
	}

	return nil
}

// logExecution stores an execution log entry
func (te *TradeExecutor) logExecution(ctx context.Context, runID uuid.UUID, level, message string) {
	log := &database.ExecutionLog{
		ID:        uuid.New(),
		RunID:     runID,
		Level:     level,
		Message:   message,
		Metadata:  database.LogMetadata{},
		Timestamp: time.Now(),
	}

	if err := te.repo.CreateLog(ctx, log); err != nil {
		te.logger.Error("Failed to store execution log", zap.Error(err))
	}
}
