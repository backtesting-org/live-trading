package real_time

import (
	"time"

	"github.com/shopspring/decimal"
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
	Price    decimal.Decimal
	Quantity decimal.Decimal
}

// TradeMessage represents a parsed trade update from WebSocket
type TradeMessage struct {
	Coin      string
	Price     decimal.Decimal
	Quantity  decimal.Decimal
	Side      string
	Timestamp time.Time
	Hash      string
	TradeID   int64
}

// PositionMessage represents a parsed position update from WebSocket
type PositionMessage struct {
	Coin           string
	Size           decimal.Decimal
	EntryPrice     decimal.Decimal
	MarkPrice      decimal.Decimal
	LiquidationPx  decimal.Decimal
	UnrealizedPnl  decimal.Decimal
	Leverage       int
	MarginUsed     decimal.Decimal
	PositionValue  decimal.Decimal
	ReturnOnEquity decimal.Decimal
	Timestamp      time.Time
}

// AccountBalanceMessage represents a parsed account balance update from WebSocket
type AccountBalanceMessage struct {
	TotalValue        decimal.Decimal
	AvailableBalance  decimal.Decimal
	Withdrawable      decimal.Decimal
	TotalMarginUsed   decimal.Decimal
	TotalNtlPos       decimal.Decimal
	TotalRawUsd       decimal.Decimal
	TotalAccountValue decimal.Decimal
	Timestamp         time.Time
}

// KlineMessage represents a parsed kline/candlestick update from WebSocket
type KlineMessage struct {
	Coin      string
	Interval  string
	OpenTime  time.Time
	CloseTime time.Time
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    decimal.Decimal
	Timestamp time.Time
}
