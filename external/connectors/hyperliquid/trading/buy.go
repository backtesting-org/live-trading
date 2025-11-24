package trading

import (
	"context"

	"github.com/sonirico/go-hyperliquid"
)

func (t *TradingService) PlaceBuyLimitOrder(coin string, size, price float64) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true, nil)
}

func (t *TradingService) PlaceBuyMarketOrder(coin string, size, slippage float64) (hyperliquid.OrderStatus, error) {
	return t.exchange.MarketOpen(context.Background(), coin, true, size, nil, slippage, nil, nil)
}

func (t *TradingService) PlaceBuyStopLoss(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, true, "sl", true)
}

func (t *TradingService) PlaceBuyTakeProfit(coin string, size, triggerPrice float64) (hyperliquid.OrderStatus, error) {
	return t.placeTriggerOrder(coin, size, triggerPrice, true, "tp", true)
}

func (t *TradingService) PlaceBuyLimitOrderWithCustomRef(coin string, size, price float64, customRef string) (hyperliquid.OrderStatus, error) {
	return t.placeLimitOrder(coin, size, price, true, &customRef)
}
