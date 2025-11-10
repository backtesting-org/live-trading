package handlers

import (
	"net/http"

	"github.com/backtesting-org/live-trading/internal/exchanges/paradex"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OrdersHandler struct {
	paradexConnector *paradex.Paradex
	logger           *zap.Logger
}

func NewOrdersHandler(paradexConnector *paradex.Paradex, logger *zap.Logger) *OrdersHandler {
	return &OrdersHandler{
		paradexConnector: paradexConnector,
		logger:           logger,
	}
}

// GetOrder retrieves order details from Paradex
// GET /api/orders/:order_id
func (h *OrdersHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	// Get order from Paradex
	order, err := h.paradexConnector.GetRawOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order", zap.String("order_id", orderID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

// GetOpenOrders retrieves all open orders
// GET /api/orders/open
func (h *OrdersHandler) GetOpenOrders(c *gin.Context) {
	market := c.Query("market") // Optional market filter

	var marketPtr *string
	if market != "" {
		marketPtr = &market
	}

	orders, err := h.paradexConnector.GetRawOpenOrders(c.Request.Context(), marketPtr)
	if err != nil {
		h.logger.Error("Failed to get open orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
	})
}

// GetAccountSummary retrieves account summary
// GET /api/account/summary
func (h *OrdersHandler) GetAccountSummary(c *gin.Context) {
	summary, err := h.paradexConnector.GetAccountSummary(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get account summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetBalances retrieves account balances
// GET /api/account/balances
func (h *OrdersHandler) GetBalances(c *gin.Context) {
	balances, err := h.paradexConnector.GetBalances(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get balances", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balances": balances,
	})
}

// GetPositions retrieves current positions
// GET /api/account/positions
func (h *OrdersHandler) GetPositions(c *gin.Context) {
	positions, err := h.paradexConnector.GetUserPositions(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get positions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"positions": positions,
	})
}

// GetTradeHistory retrieves trade history
// GET /api/account/trades
func (h *OrdersHandler) GetTradeHistory(c *gin.Context) {
	market := c.Query("market") // Optional market filter

	var marketPtr *string
	if market != "" {
		marketPtr = &market
	}

	trades, err := h.paradexConnector.GetTradeHistory(c.Request.Context(), marketPtr)
	if err != nil {
		h.logger.Error("Failed to get trade history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trades)
}
