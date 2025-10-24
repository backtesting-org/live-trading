package paradex

import (
	"context"
	"fmt"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/requests"
	"github.com/shopspring/decimal"
	"time"
)

func (p *Paradex) PlaceLimitOrder(symbol string, side connector.OrderSide, quantity, price decimal.Decimal) (*connector.OrderResponse, error) {
	orderReq := requests.PlaceOrderParams{
		Market:    symbol + "-USD-PERP",
		Side:      string(side), // "BUY" or "SELL"
		Size:      quantity.String(),
		Price:     price.String(),
		OrderType: "LIMIT",
		ClientID:  "", // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeLimit,
		Quantity:  quantity,
		Price:     price,
		FilledQty: decimal.Zero, // Update if fill info is available
		Timestamp: time.Now(),   // Or resp.CreatedAt if available
	}, nil
}

func (p *Paradex) PlaceMarketOrder(symbol string, side connector.OrderSide, quantity decimal.Decimal) (*connector.OrderResponse, error) {
	orderReq := requests.PlaceOrderParams{
		Market:    symbol,
		Side:      string(side),
		Size:      quantity.String(),
		OrderType: "MARKET",
		Price:     "0", // Market orders may still need a price field
		ClientID:  "",  // Optional
	}

	resp, err := p.paradexService.PlaceOrder(p.ctx, orderReq)
	if err != nil {
		return nil, err
	}

	return &connector.OrderResponse{
		OrderID:   resp.ID,
		Symbol:    resp.Market,
		Status:    connector.OrderStatusNew,
		Side:      side,
		Type:      connector.OrderTypeMarket,
		Quantity:  quantity,
		FilledQty: decimal.Zero,
		Timestamp: time.Now(),
	}, nil
}

func (p *Paradex) CancelOrder(symbol, orderID string) (*connector.CancelResponse, error) {
	p.tradingLogger.OrderLifecycle("Cancelling order %s for symbol %s", orderID, symbol)
	err := p.paradexService.CancelOrder(p.ctx, orderID)
	if err != nil {
		p.tradingLogger.OrderLifecycle("Failed to cancel order %s for symbol %s: %v", orderID, symbol, err)
		return nil, err
	}

	return &connector.CancelResponse{
		OrderID: orderID,
		Symbol:  symbol,
		Status:  connector.OrderCancellationRequested,
	}, nil
}

func (p *Paradex) GetOpenOrders() ([]connector.Order, error) {
	ctx := context.Background()
	paradexOrders, err := p.paradexService.GetOpenOrders(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders from Paradex: %w", err)
	}

	// Convert Paradex orders to connector format
	orders := make([]connector.Order, 0, len(paradexOrders))

	for _, paradexOrder := range paradexOrders {
		if paradexOrder == nil {
			continue
		}

		convertedOrder := p.convertParadexOrder(paradexOrder)
		orders = append(orders, convertedOrder)
	}

	p.appLogger.Debug("Retrieved %d open orders from Paradex", len(orders))
	return orders, nil
}

func (p *Paradex) GetOrderStatus(orderID string) (*connector.Order, error) {
	order, err := p.paradexService.GetOrder(p.ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order == nil {
		return nil, fmt.Errorf("order %s not found", orderID)
	}

	// Convert Paradex order to connector format
	convertedOrder := p.convertParadexOrder(order)

	return &convertedOrder, nil
}

func (p *Paradex) GetTradingHistory(symbol string, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("trading history not needed for MM strategy")
}
