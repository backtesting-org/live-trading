package paradex

import (
	"fmt"
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
	"time"
)

func (p *Paradex) FetchPrice(symbol string) (*connector.Price, error) {
	price, err := p.paradexService.GetPrice(p.ctx, symbol)

	if err != nil {
		p.appLogger.Error("Failed to fetch price", "symbol", symbol, "error", err)
		return nil, fmt.Errorf("failed to fetch price for %s: %w", symbol, err)
	}

	if price == nil {
		p.appLogger.Error("Price not found", "symbol", symbol)
		return nil, fmt.Errorf("price not found for %s", symbol)
	}

	priceValue, err := decimal.NewFromString(price.Bid)
	if err != nil {
		p.appLogger.Error("Invalid price format", "symbol", symbol, "price", price.Bid, "error", err)
		return nil, fmt.Errorf("invalid price format for %s: %w", symbol, err)
	}

	return &connector.Price{
		Symbol:    symbol,
		Price:     priceValue,
		BidPrice:  priceValue,
		AskPrice:  priceValue,
		Volume24h: decimal.Zero, // Volume not provided by Paradex
		Change24h: decimal.Zero, // Change not provided by Paradex
		Source:    p.GetConnectorInfo().Name,
		Timestamp: time.Now(), // Use current time as Paradex does not provide timestamp
	}, nil

}

func (p *Paradex) FetchOrderBook(symbol portfolio.Asset, instrument connector.Instrument, depth int) (*connector.OrderBook, error) {
	if instrument != connector.TypePerpetual {
		return nil, fmt.Errorf("order book only supported for perpetual contracts")
	}

	symbolStr := p.GetPerpSymbol(symbol)
	depthInt := int64(depth)

	orderBook, err := p.paradexService.GetOrderBook(p.ctx, symbolStr, &depthInt)
	if err != nil {
		return nil, err
	}

	// Convert bids - each bid is []string where [0] is price, [1] is size
	bids := make([]connector.PriceLevel, len(orderBook.Bids))
	for i, bid := range orderBook.Bids {
		if len(bid) >= 2 {
			price, _ := decimal.NewFromString(bid[0]) // First element is price
			size, _ := decimal.NewFromString(bid[1])  // Second element is size
			bids[i] = connector.PriceLevel{Price: price, Quantity: size}
		}
	}

	// Convert asks - each ask is []string where [0] is price, [1] is size
	asks := make([]connector.PriceLevel, len(orderBook.Asks))
	for i, ask := range orderBook.Asks {
		if len(ask) >= 2 {
			price, _ := decimal.NewFromString(ask[0]) // First element is price
			size, _ := decimal.NewFromString(ask[1])  // Second element is size
			asks[i] = connector.PriceLevel{Price: price, Quantity: size}
		}
	}

	// Use the actual timestamp from Paradex if available
	timestamp := time.Now()
	if orderBook.LastUpdatedAt > 0 {
		timestamp = time.UnixMilli(orderBook.LastUpdatedAt)
	}

	return &connector.OrderBook{
		Asset:     symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}, nil
}

func (p *Paradex) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("klines not needed for MM strategy")
}

func (p *Paradex) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	return nil, fmt.Errorf("klines not needed for MM strategy")
}

func (p *Paradex) FetchRiskFundBalance(symbol string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("risk fund balance not needed for MM strategy")
}

func (p *Paradex) FetchContracts() ([]connector.ContractInfo, error) {
	return nil, fmt.Errorf("contracts not needed for MM strategy")
}
