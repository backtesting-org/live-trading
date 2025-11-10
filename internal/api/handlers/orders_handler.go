package handlers

import (
	"context"
	"net/http"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ExchangeOrderAPI defines exchange-specific order/account API methods
type ExchangeOrderAPI interface {
	GetRawOrder(ctx context.Context, orderID string) (interface{}, error)
	GetRawOpenOrders(ctx context.Context, market *string) (interface{}, error)
	GetAccountSummary(ctx context.Context) (interface{}, error)
	GetBalances(ctx context.Context) (interface{}, error)
	GetUserPositions(ctx context.Context) (interface{}, error)
	GetTradeHistory(ctx context.Context, market *string) (interface{}, error)
}

type OrdersHandler struct {
	connector connector.Connector
	logger    *zap.Logger
}

func NewOrdersHandler(conn connector.Connector, logger *zap.Logger) *OrdersHandler {
	return &OrdersHandler{
		connector: conn,
		logger:    logger,
	}
}

// getOrderAPI attempts to get exchange-specific order API from connector
func (h *OrdersHandler) getOrderAPI() (ExchangeOrderAPI, error) {
	if api, ok := h.connector.(ExchangeOrderAPI); ok {
		return api, nil
	}
	return nil, nil // Exchange doesn't support these endpoints
}

// GetOrder retrieves order details from exchange
// GET /api/orders/:order_id
func (h *OrdersHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("order_id")

	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support order API"})
		return
	}

	order, err := api.GetRawOrder(c.Request.Context(), orderID)
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

	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support order API"})
		return
	}

	orders, err := api.GetRawOpenOrders(c.Request.Context(), marketPtr)
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
	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support account API"})
		return
	}

	summary, err := api.GetAccountSummary(c.Request.Context())
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
	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support balances API"})
		return
	}

	balances, err := api.GetBalances(c.Request.Context())
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
	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support positions API"})
		return
	}

	positions, err := api.GetUserPositions(c.Request.Context())
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

	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support trade history API"})
		return
	}

	trades, err := api.GetTradeHistory(c.Request.Context(), marketPtr)
	if err != nil {
		h.logger.Error("Failed to get trade history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trades)
}

// GetSubAccounts retrieves sub-accounts
// GET /api/account/sub-accounts
func (h *OrdersHandler) GetSubAccounts(c *gin.Context) {
	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support sub-accounts API"})
		return
	}

	// Type assert to check if exchange supports GetSubAccounts
	type SubAccountsAPI interface {
		GetSubAccounts(context.Context) (interface{}, error)
	}

	if subAccountsAPI, ok := api.(SubAccountsAPI); ok {
		subAccounts, err := subAccountsAPI.GetSubAccounts(c.Request.Context())
		if err != nil {
			h.logger.Error("Failed to get sub-accounts", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, subAccounts)
	} else {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support sub-accounts"})
	}
}

// GetAccountInfo retrieves account information
// GET /api/account/info
func (h *OrdersHandler) GetAccountInfo(c *gin.Context) {
	api, err := h.getOrderAPI()
	if err != nil || api == nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support account info API"})
		return
	}

	// Type assert to check if exchange supports GetAccountInfo
	type AccountInfoAPI interface {
		GetAccountInfo(context.Context) (interface{}, error)
	}

	if accountInfoAPI, ok := api.(AccountInfoAPI); ok {
		accountInfo, err := accountInfoAPI.GetAccountInfo(c.Request.Context())
		if err != nil {
			h.logger.Error("Failed to get account info", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, accountInfo)
	} else {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Exchange does not support account info"})
	}
}
