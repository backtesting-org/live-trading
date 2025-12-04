package rest

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/adaptors"
	hyperliquid "github.com/sonirico/go-hyperliquid"
)

// TradingService interface for trading operations
type TradingService interface {
	ModifyOrder(orderID int64, coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error)
	PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error)

	// Buy operations
	PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)
	PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)
	PlaceBuyStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceBuyTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceBuyLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error)

	// Sell operations
	PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error)
	PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error)
	PlaceSellStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceSellTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error)
	PlaceSellLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error)

	// Close operations
	ClosePosition(coin string, size *float64, slippage float64) (hyperliquid.OrderStatus, error)
	CloseEntirePosition(coin string, slippage float64) (hyperliquid.OrderStatus, error)

	// Cancel operations
	CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelResponse], error)
	CancelOrderByCustomRef(coin, customRef string) (*hyperliquid.APIResponse[hyperliquid.CancelResponse], error)
}

// tradingService implementation
type tradingService struct {
	client adaptors.ExchangeClient
}

// NewTradingService creates a new trading service
func NewTradingService(client adaptors.ExchangeClient) TradingService {
	return &tradingService{client: client}
}

func (t *tradingService) ModifyOrder(orderID int64, coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	oid := &orderID
	req := hyperliquid.ModifyOrderRequest{
		Oid: oid,
		Order: hyperliquid.CreateOrderRequest{
			Coin:       coin,
			IsBuy:      isBuy,
			Price:      price,
			Size:       size,
			ReduceOnly: false,
			OrderType: hyperliquid.OrderType{
				Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
			},
		},
	}
	return ex.ModifyOrder(req)
}

func (t *tradingService) PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return nil, fmt.Errorf("exchange not configured: %w", err)
	}
	return ex.BulkOrders(orders, nil)
}
