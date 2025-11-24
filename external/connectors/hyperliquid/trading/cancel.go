package trading

import "github.com/sonirico/go-hyperliquid"

func (t *TradingService) CancelOrderByID(coin string, orderID int64) (*hyperliquid.APIResponse[hyperliquid.CancelResponse], error) {
	return t.exchange.Cancel(coin, orderID)
}

func (t *TradingService) CancelOrderByCustomRef(coin, customRef string) (*hyperliquid.APIResponse[hyperliquid.CancelResponse], error) {
	return t.exchange.CancelByCloid(coin, customRef)
}
