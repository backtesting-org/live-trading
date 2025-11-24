package hyperliquid

import (
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

// FetchKlines retrieves historical candlestick data with decimal precision
func (h *hyperliquid) FetchKlines(symbol, interval string, limit int) ([]connector.Kline, error) {
	hlInterval := convertInterval(interval)
	endTime := time.Now().Unix()
	startTime := endTime - int64(limit*intervalToSeconds(hlInterval))

	candles, err := h.marketData.GetCandles(symbol, hlInterval, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
	}

	klines := make([]connector.Kline, 0, len(candles))
	for _, candle := range candles {
		open, err := decimal.NewFromString(candle.Open)
		if err != nil {
			return nil, fmt.Errorf("invalid open price: %w", err)
		}

		high, err := decimal.NewFromString(candle.High)
		if err != nil {
			return nil, fmt.Errorf("invalid high price: %w", err)
		}

		low, err := decimal.NewFromString(candle.Low)
		if err != nil {
			return nil, fmt.Errorf("invalid low price: %w", err)
		}

		closeVal, err := decimal.NewFromString(candle.Close)
		if err != nil {
			return nil, fmt.Errorf("invalid close price: %w", err)
		}

		volume, err := decimal.NewFromString(candle.Volume)
		if err != nil {
			return nil, fmt.Errorf("invalid volume: %w", err)
		}

		klines = append(klines, connector.Kline{
			OpenTime:  time.Unix(candle.Time/1000, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeVal,
			Volume:    volume,
			CloseTime: time.Unix(candle.Timestamp/1000, 0),
		})
	}

	return klines, nil
}

// FetchPrice retrieves current price with decimal precision
func (h *hyperliquid) FetchPrice(symbol string) (*connector.Price, error) {
	mids, err := h.marketData.GetAllMids()
	if err != nil {
		return nil, fmt.Errorf("failed to get current prices: %w", err)
	}

	priceStr, exists := mids[symbol]
	if !exists {
		return nil, fmt.Errorf("price not found for symbol: %s", symbol)
	}

	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		return nil, fmt.Errorf("invalid price format for %s: %w", symbol, err)
	}

	return &connector.Price{
		Symbol:    symbol,
		Price:     price,
		Source:    "Hyperliquid",
		Timestamp: time.Now(),
	}, nil
}

// FetchOrderBook retrieves order book with decimal precision
func (h *hyperliquid) FetchOrderBook(symbol portfolio.Asset, instrument connector.Instrument, depth int) (*connector.OrderBook, error) {
	l2Book, err := h.marketData.GetL2Book(symbol.Symbol())

	if instrument != connector.TypePerpetual {
		l2Book, err = h.marketData.GetL2Book(symbol.Symbol() + "-USD")
	} else {
		l2Book, err = h.marketData.GetL2Book(h.GetPerpSymbol(symbol))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	orderBook := &connector.OrderBook{
		Asset:     symbol,
		Timestamp: time.Now(),
		Bids:      make([]connector.PriceLevel, 0, depth),
		Asks:      make([]connector.PriceLevel, 0, depth),
	}

	if l2Book.Levels == nil || len(l2Book.Levels) < 2 {
		return orderBook, nil
	}

	// Process bids (buy orders)
	for i, level := range l2Book.Levels[0] {
		if i >= depth {
			break
		}

		price := decimal.NewFromFloat(level.Px)
		quantity := decimal.NewFromFloat(level.Sz)

		orderBook.Bids = append(orderBook.Bids, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Process asks (sell orders)
	for i, level := range l2Book.Levels[1] {
		if i >= depth {
			break
		}

		price := decimal.NewFromFloat(level.Px)
		quantity := decimal.NewFromFloat(level.Sz)

		orderBook.Asks = append(orderBook.Asks, connector.PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return orderBook, nil
}

func (h *hyperliquid) GetPerpSymbol(symbol portfolio.Asset) string {
	return symbol.Symbol() + "-USD"

}

// FetchRecentTrades retrieves recent trades for the specified symbol
func (h *hyperliquid) FetchRecentTrades(symbol string, limit int) ([]connector.Trade, error) {
	return nil, fmt.Errorf("FetchRecentTrades not yet implemented for Hyperliquid")
}

// FetchRiskFundBalance retrieves risk fund balance for the specified symbol
func (h *hyperliquid) FetchRiskFundBalance(symbol string) (*connector.RiskFundBalance, error) {
	return nil, fmt.Errorf("FetchRiskFundBalance not implemented for Hyperliquid")
}

// FetchContracts retrieves available contract information
func (h *hyperliquid) FetchContracts() ([]connector.ContractInfo, error) {
	return nil, fmt.Errorf("FetchContracts not implemented for Hyperliquid")
}
