package websockets

func (s *service) OrderbookUpdates() <-chan OrderbookUpdate {
	return s.orderbookChan
}

func (s *service) TradeUpdates() <-chan TradeUpdate {
	return s.tradeChan
}

func (s *service) AccountUpdates() <-chan AccountUpdate {
	return s.accountChan
}

func (s *service) KlineUpdates() <-chan KlineUpdate {
	if s.klineBuilder == nil {
		ch := make(chan KlineUpdate)
		close(ch)
		return ch
	}
	return s.klineBuilder.Output()
}
