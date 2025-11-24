package trading

import (
	"context"

	"github.com/sonirico/go-hyperliquid"
)

func (t *TradingService) placeLimitOrder(coin string, size, price float64, isBuy bool, clientOrderID *string) (hyperliquid.OrderStatus, error) {
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

	return t.exchange.Order(context.Background(), req, nil)
}

func (t *TradingService) placeTriggerOrder(coin string, size, triggerPrice float64, isBuy bool, tpsl string, isMarket bool) (hyperliquid.OrderStatus, error) {
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

	return t.exchange.Order(context.Background(), req, nil)
}
