package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/shopspring/decimal"
)

func (p *Paradex) OrderBookUpdates() <-chan connector.OrderBook {
	if p.wsService == nil {
		ch := make(chan connector.OrderBook)
		close(ch)
		return ch
	}

	convertedChan := make(chan connector.OrderBook, 100)
	go p.convertOrderBookUpdates(convertedChan)
	return convertedChan
}

func (p *Paradex) TradeUpdates() <-chan connector.Trade {
	if p.wsService == nil {
		ch := make(chan connector.Trade)
		close(ch)
		return ch
	}

	convertedChan := make(chan connector.Trade, 100)
	go p.convertTradeUpdates(convertedChan)
	return convertedChan
}

func (p *Paradex) PositionUpdates() <-chan connector.Position {
	if p.wsService == nil {
		ch := make(chan connector.Position)
		close(ch)
		return ch
	}

	// TODO: Implement actual position updates conversion
	convertedChan := make(chan connector.Position, 100)
	go p.convertPositionUpdates(convertedChan)
	return convertedChan
}

func (p *Paradex) AccountBalanceUpdates() <-chan connector.AccountBalance {
	if p.wsService == nil {
		ch := make(chan connector.AccountBalance)
		close(ch)
		return ch
	}

	// TODO: Implement actual account balance updates conversion
	convertedChan := make(chan connector.AccountBalance, 100)
	go p.convertAccountBalanceUpdates(convertedChan)
	return convertedChan
}

func (p *Paradex) KlineUpdates() <-chan connector.Kline {
	if p.wsService == nil {
		ch := make(chan connector.Kline)
		close(ch)
		return ch
	}

	// TODO: Implement actual kline updates conversion
	convertedChan := make(chan connector.Kline, 100)
	go p.convertKlineUpdates(convertedChan)
	return convertedChan
}

func (p *Paradex) ErrorChannel() <-chan error {
	if p.wsService == nil {
		ch := make(chan error)
		close(ch)
		return ch
	}

	return p.wsService.ErrorChannel()
}

// Stub converter methods - implement these when you need the functionality
func (p *Paradex) convertPositionUpdates(out chan<- connector.Position) {
	defer close(out)
	// TODO: Convert from Paradex position format to connector.Position
	// For now, just a placeholder that doesn't send anything
	<-p.wsContext.Done()
}

func (p *Paradex) convertAccountBalanceUpdates(out chan<- connector.AccountBalance) {
	defer close(out)
	// TODO: Convert from Paradex balance format to connector.AccountBalance
	// For now, just a placeholder that doesn't send anything
	<-p.wsContext.Done()
}

func (p *Paradex) convertKlineUpdates(out chan<- connector.Kline) {
	defer close(out)

	for {
		select {
		case <-p.wsContext.Done():
			return
		case paradexKline, ok := <-p.wsService.KlineUpdates():
			if !ok {
				return
			}

			// Convert from Paradex KlineUpdate to connector.Kline
			connectorKline := connector.Kline{
				Symbol:      paradexKline.Symbol,
				Interval:    paradexKline.Interval,
				OpenTime:    paradexKline.OpenTime,
				Open:        paradexKline.Open,
				High:        paradexKline.High,
				Low:         paradexKline.Low,
				Close:       paradexKline.Close,
				Volume:      paradexKline.Volume,
				CloseTime:   paradexKline.CloseTime,
				QuoteVolume: decimal.Zero,                 // Not available from Paradex
				TradeCount:  int(paradexKline.TradeCount), // Convert int64 to int
				TakerVolume: decimal.Zero,                 // Not available from Paradex
			}

			select {
			case out <- connectorKline:
			case <-p.wsContext.Done():
				return
			default:
				// Channel full, drop update to prevent blocking
				p.appLogger.Debug("Dropped kline update for %s due to full channel", paradexKline.Symbol)
			}
		}
	}
}
