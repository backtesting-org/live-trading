package websockets

func (s *Service) OrderbookUpdates() <-chan OrderbookUpdate {
	return s.orderbookChan
}

func (s *Service) TradeUpdates() <-chan TradeUpdate {
	return s.tradeChan
}

func (s *Service) AccountUpdates() <-chan AccountUpdate {
	return s.accountChan
}

func (s *Service) KlineUpdates() <-chan KlineUpdate {
	if s.klineBuilder == nil {
		ch := make(chan KlineUpdate)
		close(ch)
		return ch
	}
	return s.klineBuilder.Output()
}
