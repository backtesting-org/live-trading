package trading

import (
	"context"

	"github.com/sonirico/go-hyperliquid"
)

func (t *TradingService) PlaceSellLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false, nil)
}

func (t *TradingService) PlaceSellMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	return t.exchange.MarketOpen(context.Background(), coin, false, size, nil, slippage, nil, nil)
}

func (t *TradingService) PlaceSellStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, false, "sl", true)
}

func (t *TradingService) PlaceSellTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, false, "tp", true)
}

func (t *TradingService) PlaceSellLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, false, &customRef)
}
