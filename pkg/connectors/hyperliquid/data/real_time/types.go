package real_time

import (
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/kronos/numerical"
)

// OrderBookMessage represents a parsed L2 order book update from WebSocket
type OrderBookMessage struct {
	Coin      string
	Timestamp time.Time
	Bids      []PriceLevel
	Asks      []PriceLevel
}

// PriceLevel represents a single price level in the order book
type PriceLevel struct {
	Price    numerical.Decimal
	Quantity numerical.Decimal
}

// TradeMessage represents a parsed trade update from WebSocket
type TradeMessage struct {
	Coin      string
	Price     numerical.Decimal
	Quantity  numerical.Decimal
	Side      string
	Timestamp time.Time
	Hash      string
	TradeID   int64
}

// PositionMessage represents a parsed position update from WebSocket
type PositionMessage struct {
	Coin           string
	Size           numerical.Decimal
	EntryPrice     numerical.Decimal
	MarkPrice      numerical.Decimal
	LiquidationPx  numerical.Decimal
	UnrealizedPnl  numerical.Decimal
	Leverage       int
	MarginUsed     numerical.Decimal
	PositionValue  numerical.Decimal
	ReturnOnEquity numerical.Decimal
	Timestamp      time.Time
}

// AccountBalanceMessage represents a parsed account balance update from WebSocket
type AccountBalanceMessage struct {
	TotalValue        numerical.Decimal
	AvailableBalance  numerical.Decimal
	Withdrawable      numerical.Decimal
	TotalMarginUsed   numerical.Decimal
	TotalNtlPos       numerical.Decimal
	TotalRawUsd       numerical.Decimal
	TotalAccountValue numerical.Decimal
	Timestamp         time.Time
}

// KlineMessage represents a parsed kline/candlestick update from WebSocket
type KlineMessage struct {
	Coin      string
	Interval  string
	OpenTime  time.Time
	CloseTime time.Time
	Open      numerical.Decimal
	High      numerical.Decimal
	Low       numerical.Decimal
	Close     numerical.Decimal
	Volume    numerical.Decimal
	Timestamp time.Time
}
