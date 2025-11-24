package hyperliquid

import (
	"fmt"
	"strconv"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/shopspring/decimal"
)

// PlaceLimitOrder places a limit order on Hyperliquid
func (h *Hyperliquid) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price decimal.Decimal) (*connector.OrderResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	var result interface{}
	var err error

	if side == connector.OrderSideBuy {
		result, err = h.trading.PlaceBuyLimitOrder(symbol, quantity.InexactFloat64(), price.InexactFloat64())
	} else {
		result, err = h.trading.PlaceSellLimitOrder(symbol, quantity.InexactFloat64(), price.InexactFloat64())
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s limit order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   extractOrderID(result),
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		Timestamp: time.Now(),
	}, nil
}

// PlaceMarketOrder places a market order on Hyperliquid
func (h *Hyperliquid) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity decimal.Decimal) (*connector.OrderResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	var result interface{}
	var err error
	slippage := 0.05 // 5% default slippage

	if side == connector.OrderSideBuy {
		result, err = h.trading.PlaceBuyMarketOrder(symbol, quantity.InexactFloat64(), slippage)
	} else {
		result, err = h.trading.PlaceSellMarketOrder(symbol, quantity.InexactFloat64(), slippage)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to place %s market order: %w", side, err)
	}

	return &connector.OrderResponse{
		OrderID:   extractOrderID(result),
		Symbol:    symbol,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		Timestamp: time.Now(),
	}, nil
}

// CancelOrder cancels an existing order on Hyperliquid
func (h *Hyperliquid) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	if !h.SupportsTradingOperations() {
		return nil, fmt.Errorf("trading operations not supported")
	}

	oid, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID format: %w", err)
	}

	_, err = h.trading.CancelOrderByID(symbol, oid)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &connector.CancelResponse{
		OrderID:   orderID,
		Symbol:    symbol,
		Status:    connector.OrderStatusCanceled,
		Timestamp: time.Now(),
	}, nil
}

// GetOpenOrders retrieves current open orders
func (h *Hyperliquid) GetOpenOrders() ([]connector.Order, error) {
	orders, err := h.marketData.GetOpenOrders(h.config.AccountAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	var connectorOrders []connector.Order
	for _, order := range orders {
		connectorOrder := connector.Order{
			ID:        fmt.Sprintf("%d", order.Oid),
			Symbol:    order.Coin,
			Side:      connector.FromString(order.Side),
			Quantity:  decimal.NewFromFloat(order.Size),
			Price:     decimal.NewFromFloat(order.LimitPx),
			CreatedAt: time.Unix(order.Timestamp/1000, 0),
		}
		connectorOrders = append(connectorOrders, connectorOrder)
	}

	return connectorOrders, nil
}

// GetOrderStatus retrieves the status of a specific order
func (h *Hyperliquid) GetOrderStatus(orderID string) (*connector.Order, error) {
	return nil, fmt.Errorf("GetOrderStatus not yet implemented for Hyperliquid")
}
