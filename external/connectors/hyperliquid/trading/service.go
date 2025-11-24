package trading

import (
	"context"

	hyperliquid "github.com/sonirico/go-hyperliquid"
)

type TradingService struct {
	exchange *hyperliquid.Exchange
}

func NewTradingService(exchange *hyperliquid.Exchange) *TradingService {
	return &TradingService{exchange: exchange}
}

func (t *TradingService) ModifyOrder(orderID int64, coin string, size, price float64, isBuy bool) (hyperliquid.OrderStatus, error) {
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
	return t.exchange.ModifyOrder(context.Background(), req)
}

func (t *TradingService) PlaceBulkOrders(orders []hyperliquid.CreateOrderRequest) (*hyperliquid.APIResponse[hyperliquid.OrderResponse], error) {
	return t.exchange.BulkOrders(context.Background(), orders, nil)
}
