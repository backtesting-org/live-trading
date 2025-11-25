package bybit

import "github.com/backtesting-org/kronos-sdk/pkg/types/connector"

func (b *bybit) AccountBalanceUpdates() <-chan connector.AccountBalance {
	return b.balanceCh
}

func (b *bybit) PositionUpdates() <-chan connector.Position {
	return b.positionCh
}

func (b *bybit) TradeUpdates() <-chan connector.Trade {
	return b.tradeCh
}

func (b *bybit) OrderBookUpdates() <-chan connector.OrderBook {
	return b.orderBookCh
}

func (b *bybit) KlineUpdates() <-chan connector.Kline {
	return b.klineCh
}

func (b *bybit) ErrorChannel() <-chan error {
	return b.errorCh
}

func (b *bybit) ErrorUpdates() <-chan error {
	return b.errorCh
}

// IsWebSocketConnected returns whether the WebSocket is connected
func (b *bybit) IsWebSocketConnected() bool {
	return b.initialized && b.realTime != nil
}
