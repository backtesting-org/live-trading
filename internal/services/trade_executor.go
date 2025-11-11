package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TradeExecutor executes trading signals on exchanges
type TradeExecutor struct {
	connector       connector.Connector
	positionManager *PositionManager
	repo            *database.Repository
	eventBus        *EventBus
	logger          *zap.Logger
	timeProvider    temporal.TimeProvider
}

// NewTradeExecutor creates a new trade executor service
func NewTradeExecutor(
	conn connector.Connector,
	positionManager *PositionManager,
	repo *database.Repository,
	eventBus *EventBus,
	logger *zap.Logger,
	timeProvider temporal.TimeProvider,
) *TradeExecutor {
	return &TradeExecutor{
		connector:       conn,
		positionManager: positionManager,
		repo:            repo,
		eventBus:        eventBus,
		logger:          logger,
		timeProvider:    timeProvider,
	}
}

// ExecuteSignal processes a trading signal and executes it on the exchange
func (te *TradeExecutor) ExecuteSignal(ctx context.Context, runID uuid.UUID, signal *strategy.Signal) error {
	// Check if connector is available
	if te.connector == nil {
		return fmt.Errorf("connector not available - cannot execute trades")
	}

	te.logger.Info("Executing signal",
		zap.String("run_id", runID.String()),
		zap.String("signal_id", signal.ID.String()),
		zap.String("strategy", string(signal.Strategy)))

	// Process each trade action in the signal and collect order IDs
	orderIDs := make(map[int]string) // Map action index to order ID
	for i, action := range signal.Actions {
		orderID, err := te.executeTradeAction(ctx, runID, signal.ID, action)
		if err != nil {
			te.logger.Error("Failed to execute trade action",
				zap.String("signal_id", signal.ID.String()),
				zap.String("asset", action.Asset.Symbol()),
				zap.Error(err))

			// Store error in execution logs
			te.logExecution(ctx, runID, "error", fmt.Sprintf("Failed to execute %s on %s: %v",
				action.Action, action.Asset.Symbol(), err))

			return err
		}
		orderIDs[i] = orderID
	}

	// Store signal in database with order IDs
	if err := te.storeSignal(ctx, runID, signal, orderIDs); err != nil {
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

// executeTradeAction executes a single trade action and returns the order ID
func (te *TradeExecutor) executeTradeAction(
	ctx context.Context,
	runID uuid.UUID,
	signalID uuid.UUID,
	action strategy.TradeAction,
) (string, error) {
	// Check risk limits before executing
	if err := te.positionManager.CheckRiskLimits(action); err != nil {
		te.logger.Warn("Trade rejected by risk manager",
			zap.String("asset", action.Asset.Symbol()),
			zap.Error(err))
		return "", err
	}

	// Execute the trade using the connector
	orderID, err := te.executeTradeOnExchange(ctx, action)
	if err != nil {
		return "", err
	}

	// Update position manager
	te.positionManager.UpdatePosition(action, orderID, runID.String())

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

	return orderID, nil
}

// executeTradeOnExchange executes a trade on the configured exchange
func (te *TradeExecutor) executeTradeOnExchange(ctx context.Context, action strategy.TradeAction) (string, error) {
	side := connector.OrderSideBuy
	if action.Action == strategy.ActionSell {
		side = connector.OrderSideSell
	}

	// Get the exchange-specific symbol format
	symbol := te.getExchangeSymbol(action.Asset)

	// Use market order for simplicity
	resp, err := te.connector.PlaceMarketOrder(symbol, side, action.Quantity)
	if err != nil {
		return "", err
	}

	return resp.OrderID, nil
}

// getExchangeSymbol gets the exchange-specific symbol format for an asset
func (te *TradeExecutor) getExchangeSymbol(asset portfolio.Asset) string {
	// For exchanges that support the GetPerpSymbol method, use it
	if symbolGetter, ok := te.connector.(interface{ GetPerpSymbol(portfolio.Asset) string }); ok {
		return symbolGetter.GetPerpSymbol(asset)
	}

	// Default: use the asset symbol as-is
	return asset.Symbol()
}

// storeSignal saves a trading signal to the database with order IDs
func (te *TradeExecutor) storeSignal(ctx context.Context, runID uuid.UUID, signal *strategy.Signal, orderIDs map[int]string) error {
	for i, action := range signal.Actions {
		dbSignal := &database.TradingSignal{
			ID:         uuid.New(),
			RunID:      runID,
			SignalType: strings.ToUpper(string(action.Action)), // Convert to uppercase for DB constraint
			Asset:      action.Asset.Symbol(),
			Exchange:   string(action.Exchange),
			Quantity:   action.Quantity,
			Price:      action.Price,
			Timestamp:  signal.Timestamp,
			Executed:   true, // Mark as executed
		}

		// Add order ID if available
		if orderID, ok := orderIDs[i]; ok && orderID != "" {
			dbSignal.OrderID = &orderID
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
		Timestamp: te.timeProvider.Now(),
	}

	if err := te.repo.CreateLog(ctx, log); err != nil {
		te.logger.Error("Failed to store execution log", zap.Error(err))
	}
}
