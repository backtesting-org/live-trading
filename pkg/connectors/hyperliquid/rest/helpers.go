package rest

import (
	"fmt"

	"github.com/sonirico/go-hyperliquid"
)

func (t *tradingService) placeLimitOrder(coin string, size, price float64, isBuy bool, clientOrderID *string) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      price,
		Size:       size,
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Limit: &hyperliquid.LimitOrderType{Tif: hyperliquid.TifGtc},
		},
		ClientOrderID: clientOrderID,
	}

	return ex.Order(req, nil)
}

func (t *tradingService) placeTriggerOrder(coin string, size, triggerPrice float64, isBuy bool, isMarket bool) (hyperliquid.OrderStatus, error) {
	ex, err := t.client.GetExchange()
	if err != nil {
		return hyperliquid.OrderStatus{}, fmt.Errorf("exchange not configured: %w", err)
	}

	req := hyperliquid.CreateOrderRequest{
		Coin:       coin,
		IsBuy:      isBuy,
		Price:      triggerPrice,
		Size:       size,
		ReduceOnly: false,
		OrderType: hyperliquid.OrderType{
			Trigger: &hyperliquid.TriggerOrderType{
				TriggerPx: triggerPrice,
				IsMarket:  isMarket,
			},
		},
	}

	return ex.Order(req, nil)
}
