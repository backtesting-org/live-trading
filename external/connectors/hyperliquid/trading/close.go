package trading

import (
	"context"

	"github.com/sonirico/go-hyperliquid"
)

func (t *TradingService) ClosePosition(coin string, size *float64, slippage float64) (hyperliquid.OrderStatus, error) {
	return t.exchange.MarketClose(context.Background(), coin, size, nil, slippage, nil, nil)
}

func (t *TradingService) CloseEntirePosition(coin string, slippage float64) (hyperliquid.OrderStatus, error) {
	return t.ClosePosition(coin, nil, slippage)
}
