package handlers

import (
	"net/http"
	"strconv"

	"github.com/backtesting-org/live-trading/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StrategyHandler handles strategy execution endpoints
type StrategyHandler struct {
	executor *services.StrategyExecutor
	logger   *zap.Logger
}

// NewStrategyHandler creates a new strategy handler
func NewStrategyHandler(executor *services.StrategyExecutor, logger *zap.Logger) *StrategyHandler {
	return &StrategyHandler{
		executor: executor,
		logger:   logger,
	}
}

// StartStrategyRequest represents a request to start a strategy
type StartStrategyRequest struct {
	PluginID uuid.UUID  `json:"plugin_id" binding:"required"`
	ConfigID *uuid.UUID `json:"config_id,omitempty"`
}

// StartStrategy starts a strategy execution
// POST /api/v1/strategies/start
func (h *StrategyHandler) StartStrategy(c *gin.Context) {
	var req StartStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	runID, err := h.executor.StartStrategy(c.Request.Context(), req.PluginID, req.ConfigID)
	if err != nil {
		h.logger.Error("Failed to start strategy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to start strategy",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Strategy started",
		zap.String("run_id", runID.String()),
		zap.String("plugin_id", req.PluginID.String()))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Strategy started successfully",
		Data: map[string]interface{}{
			"run_id":    runID.String(),
			"plugin_id": req.PluginID.String(),
		},
	})
}

// StopStrategy stops a running strategy
// POST /api/v1/strategies/:runId/stop
func (h *StrategyHandler) StopStrategy(c *gin.Context) {
	runIDParam := c.Param("runId")
	runID, err := uuid.Parse(runIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid run ID",
			Message: "Run ID must be a valid UUID",
		})
		return
	}

	if err := h.executor.StopStrategy(c.Request.Context(), runID); err != nil {
		h.logger.Error("Failed to stop strategy", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to stop strategy",
			Message: err.Error(),
		})
		return
	}

	h.logger.Info("Strategy stopped", zap.String("run_id", runID.String()))

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Strategy stopped successfully",
		Data:    map[string]interface{}{"run_id": runID.String()},
	})
}

// GetRunStatus retrieves the status of a strategy run
// GET /api/v1/strategies/:runId/status
func (h *StrategyHandler) GetRunStatus(c *gin.Context) {
	runIDParam := c.Param("runId")
	runID, err := uuid.Parse(runIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid run ID",
			Message: "Run ID must be a valid UUID",
		})
		return
	}

	status, err := h.executor.GetRunStatus(c.Request.Context(), runID)
	if err != nil {
		h.logger.Error("Failed to get run status", zap.Error(err))
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Run not found",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Run status retrieved successfully",
		Data:    status,
	})
}

// ListRuns retrieves execution history for a plugin
// GET /api/v1/strategies/runs
func (h *StrategyHandler) ListRuns(c *gin.Context) {
	pluginIDParam := c.Query("plugin_id")
	if pluginIDParam == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing plugin_id parameter",
			Message: "plugin_id is required",
		})
		return
	}

	pluginID, err := uuid.Parse(pluginIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid plugin ID",
			Message: "Plugin ID must be a valid UUID",
		})
		return
	}

	// Parse pagination params
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	runs, err := h.executor.ListRuns(c.Request.Context(), pluginID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list runs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to list runs",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Runs retrieved successfully",
		Data: map[string]interface{}{
			"runs":   runs,
			"count":  len(runs),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetRunStats retrieves detailed statistics for a run
// GET /api/v1/strategies/:runId/stats
func (h *StrategyHandler) GetRunStats(c *gin.Context) {
	runIDParam := c.Param("runId")
	runID, err := uuid.Parse(runIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid run ID",
			Message: "Run ID must be a valid UUID",
		})
		return
	}

	stats, err := h.executor.GetRunStats(c.Request.Context(), runID)
	if err != nil {
		h.logger.Error("Failed to get run stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get run stats",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Run stats retrieved successfully",
		Data:    stats,
	})
}
