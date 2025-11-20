package handlers

import (
	"context"
	"net/http"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/stores/activity"
	"github.com/backtesting-org/live-trading/internal/database"
	"github.com/backtesting-org/live-trading/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/trishtzy/go-paradex/models"
	"go.uber.org/zap"
)

// DashboardHandler handles dashboard aggregated data endpoints
type DashboardHandler struct {
	connector        connector.Connector
	strategyExecutor *services.StrategyExecutor
	positionManager  activity.Positions
	repo             *database.Repository
	logger           *zap.Logger
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(
	conn connector.Connector,
	strategyExecutor *services.StrategyExecutor,
	positionManager activity.Positions,
	repo *database.Repository,
	logger *zap.Logger,
) *DashboardHandler {
	return &DashboardHandler{
		connector:        conn,
		strategyExecutor: strategyExecutor,
		positionManager:  positionManager,
		repo:             repo,
		logger:           logger,
	}
}

// PositionInfo represents position information for the dashboard
type PositionInfo struct {
	Market         string `json:"market"`
	Side           string `json:"side"`
	Size           string `json:"size"`
	AverageEntry   string `json:"average_entry_price"`
	UnrealizedPnL  string `json:"unrealized_pnl"`
	StrategyRunID  string `json:"strategy_run_id,omitempty"`
}

// StrategyInfo represents strategy information for the dashboard
type StrategyInfo struct {
	RunID           string         `json:"run_id"`
	PluginID        string         `json:"plugin_id"`
	Status          string         `json:"status"`
	RealizedPnL     string         `json:"realized_pnl"`
	UnrealizedPnL   string         `json:"unrealized_pnl"`
	WinRate         string         `json:"win_rate"`
	TotalTrades     int64          `json:"total_trades"`
	CurrentPosition *PositionInfo  `json:"current_position,omitempty"`
}

// DashboardStats represents the aggregated dashboard data
type DashboardStats struct {
	Account struct {
		TotalPortfolioValue string `json:"total_portfolio_value"`
		TotalPnL            string `json:"total_pnl"`
		FreeCollateral      string `json:"free_collateral"`
		SettlementAsset     string `json:"settlement_asset"`
	} `json:"account"`
	Strategies struct {
		TotalActive int             `json:"total_active"`
		Strategies  []StrategyInfo  `json:"strategies"`
	} `json:"strategies"`
	Positions struct {
		TotalActive int            `json:"total_active"`
		Positions   []PositionInfo `json:"positions"`
	} `json:"positions"`
}

// GetDashboardStats retrieves aggregated dashboard statistics
// GET /api/v1/dashboard/stats
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats := DashboardStats{}

	// 1. Get account summary from exchange
	if err := h.populateAccountData(ctx, &stats); err != nil {
		h.logger.Error("Failed to get account data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account data"})
		return
	}

	// 2. Get all running strategies
	if err := h.populateStrategyData(ctx, &stats); err != nil {
		h.logger.Error("Failed to get strategy data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get strategy data"})
		return
	}

	// 3. Get all positions from exchange
	if err := h.populatePositionData(ctx, &stats); err != nil {
		h.logger.Error("Failed to get position data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get position data"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Dashboard stats retrieved successfully",
		Data:    stats,
	})
}

func (h *DashboardHandler) populateAccountData(ctx context.Context, stats *DashboardStats) error {
	// Get account summary from exchange API
	api, ok := h.connector.(ExchangeOrderAPI)
	if !ok {
		return nil // Exchange doesn't support this API
	}

	summary, err := api.GetAccountSummary(ctx)
	if err != nil {
		return err
	}

	// Type assert to the actual Paradex response type
	if accountResp, ok := summary.(*models.ResponsesAccountSummaryResponse); ok {
		accountValue := accountResp.AccountValue
		totalCollateral := accountResp.TotalCollateral

		// Calculate total PnL (account_value - total_collateral)
		av, _ := decimal.NewFromString(accountValue)
		tc, _ := decimal.NewFromString(totalCollateral)
		totalPnL := av.Sub(tc)

		stats.Account.TotalPortfolioValue = accountValue
		stats.Account.TotalPnL = totalPnL.String()
		stats.Account.FreeCollateral = accountResp.FreeCollateral
		stats.Account.SettlementAsset = accountResp.SettlementAsset
	}

	return nil
}

func (h *DashboardHandler) populateStrategyData(ctx context.Context, stats *DashboardStats) error {
	// Get all running strategy runs from database
	runs, err := h.repo.GetActiveRuns(ctx)
	if err != nil {
		return err
	}

	stats.Strategies.TotalActive = len(runs)
	stats.Strategies.Strategies = make([]StrategyInfo, 0)

	for _, run := range runs {
		strategyInfo := StrategyInfo{
			RunID:    run.ID.String(),
			PluginID: run.PluginID.String(),
			Status:   run.Status,
		}

		// Get run stats
		runStats, err := h.strategyExecutor.GetRunStats(ctx, run.ID)
		if err == nil {
			// Extract stats
			if rp, ok := runStats["realized_pnl"].(string); ok {
				strategyInfo.RealizedPnL = rp
			}
			if up, ok := runStats["unrealized_pnl"].(string); ok {
				strategyInfo.UnrealizedPnL = up
			}
			if wr, ok := runStats["win_rate"].(string); ok {
				strategyInfo.WinRate = wr
			}
			if tt, ok := runStats["total_trades"].(int64); ok {
				strategyInfo.TotalTrades = tt
			}
		}

		// TODO: Get current position for this strategy when SDK provides GetPositionsByRunID
		// For now, skip position data

		stats.Strategies.Strategies = append(stats.Strategies.Strategies, strategyInfo)
	}

	return nil
}

func (h *DashboardHandler) populatePositionData(ctx context.Context, stats *DashboardStats) error {
	// Get positions from exchange API
	api, ok := h.connector.(ExchangeOrderAPI)
	if !ok {
		return nil // Exchange doesn't support this API
	}

	positions, err := api.GetUserPositions(ctx)
	if err != nil {
		return err
	}

	stats.Positions.Positions = make([]PositionInfo, 0)

	// Type assert to the actual Paradex response type
	if posResp, ok := positions.(*models.ResponsesGetPositionsResp); ok {
		stats.Positions.TotalActive = len(posResp.Results)

		for _, pos := range posResp.Results {
			posInfo := PositionInfo{
				Market:        pos.Market,
				Side:          pos.Side,
				Size:          pos.Size,
				AverageEntry:  pos.AverageEntryPrice,
				UnrealizedPnL: pos.UnrealizedPnl,
			}

			stats.Positions.Positions = append(stats.Positions.Positions, posInfo)
		}
	}

	// Match positions with strategy runs
	h.matchPositionsToStrategies(stats)

	return nil
}

func (h *DashboardHandler) matchPositionsToStrategies(stats *DashboardStats) {
	// TODO: Match positions to strategies when SDK provides GetAllPositions
	// For now, skip matching
}
