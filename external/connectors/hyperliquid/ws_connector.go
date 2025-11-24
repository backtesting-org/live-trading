package hyperliquid

import (
	"context"
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/data/real_time"
)

// StartWebSocket starts the WebSocket connection for real-time data
func (h *hyperliquid) StartWebSocket(ctx context.Context) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	return h.realTime.Connect(ctx)
}

// StopWebSocket stops the WebSocket connection
func (h *hyperliquid) StopWebSocket() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	return h.realTime.Disconnect()
}

// IsWebSocketConnected returns whether the WebSocket is connected
func (h *hyperliquid) IsWebSocketConnected() bool {
	return h.initialized && h.wsClient.IsConfigured()
}

// OrderBookUpdates returns a channel for order book updates
func (h *hyperliquid) OrderBookUpdates() <-chan connector.OrderBook {
	return h.orderBookCh
}

// TradeUpdates returns a channel for trade updates
func (h *hyperliquid) TradeUpdates() <-chan connector.Trade {
	return h.tradeCh
}

// PositionUpdates returns a channel for position updates
func (h *hyperliquid) PositionUpdates() <-chan connector.Position {
	return h.positionCh
}

// AccountBalanceUpdates returns a channel for account balance updates
func (h *hyperliquid) AccountBalanceUpdates() <-chan connector.AccountBalance {
	return h.balanceCh
}

// KlineUpdates returns a channel for kline/candlestick updates
func (h *hyperliquid) KlineUpdates() <-chan connector.Kline {
	return h.klineCh
}

// ErrorChannel returns a channel for WebSocket errors
func (h *hyperliquid) ErrorChannel() <-chan error {
	return h.errorCh
}

// SubscribeOrderBook subscribes to order book updates for an asset
func (h *hyperliquid) SubscribeOrderBook(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToOrderBook(asset.Symbol(), func(obMsg *real_time.OrderBookMessage) {
		bids := make([]connector.PriceLevel, len(obMsg.Bids))
		for i, bid := range obMsg.Bids {
			bids[i] = connector.PriceLevel{
				Price:    bid.Price,
				Quantity: bid.Quantity,
			}
		}

		asks := make([]connector.PriceLevel, len(obMsg.Asks))
		for i, ask := range obMsg.Asks {
			asks[i] = connector.PriceLevel{
				Price:    ask.Price,
				Quantity: ask.Quantity,
			}
		}

		orderBook := connector.OrderBook{
			Asset:     asset,
			Timestamp: obMsg.Timestamp,
			Bids:      bids,
			Asks:      asks,
		}

		select {
		case h.orderBookCh <- orderBook:
		default:
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["orderbook:"+asset.Symbol()] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (h *hyperliquid) UnsubscribeOrderBook(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	subID, exists := h.subscriptions["orderbook:"+asset.Symbol()]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for orderbook:%s", asset.Symbol())
	}
	delete(h.subscriptions, "orderbook:"+asset.Symbol())
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromOrderBook(asset.Symbol(), subID)
}

// SubscribeTrades subscribes to trade updates for an asset
func (h *hyperliquid) SubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToTrades(asset.Symbol(), func(trades []real_time.TradeMessage) {
		for _, trade := range trades {
			select {
			case h.tradeCh <- connector.Trade{
				Symbol:    trade.Coin,
				Exchange:  connector.Hyperliquid,
				Price:     trade.Price,
				Quantity:  trade.Quantity,
				Side:      connector.FromString(trade.Side),
				Timestamp: trade.Timestamp,
			}:
			default:
			}
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["trades:"+asset.Symbol()] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeTrades unsubscribes from trade updates
func (h *hyperliquid) UnsubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	subID, exists := h.subscriptions["trades:"+asset.Symbol()]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for trades:%s", asset.Symbol())
	}
	delete(h.subscriptions, "trades:"+asset.Symbol())
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromTrades(asset.Symbol(), subID)
}

// SubscribePositions subscribes to position updates
func (h *hyperliquid) SubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToPositions(h.config.AccountAddress, func(posMsg *real_time.PositionMessage) {
		if posMsg.Coin != asset.Symbol() {
			return
		}

		side := connector.OrderSideBuy
		if posMsg.Size.IsNegative() {
			side = connector.OrderSideSell
		}

		select {
		case h.positionCh <- connector.Position{
			Symbol:        asset,
			Exchange:      connector.Hyperliquid,
			Side:          side,
			Size:          posMsg.Size.Abs(),
			EntryPrice:    posMsg.EntryPrice,
			MarkPrice:     posMsg.MarkPrice,
			UnrealizedPnL: posMsg.UnrealizedPnl,
			RealizedPnL:   parseDecimal("0"),
			UpdatedAt:     posMsg.Timestamp,
		}:
		default:
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["positions:"+asset.Symbol()] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribePositions unsubscribes from position updates
func (h *hyperliquid) UnsubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	_, exists := h.subscriptions["positions:"+asset.Symbol()]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for positions:%s", asset.Symbol())
	}
	delete(h.subscriptions, "positions:"+asset.Symbol())
	h.subMu.Unlock()

	// No unsubscribe method for positions yet
	return nil
}

// SubscribeAccountBalance subscribes to account balance updates
func (h *hyperliquid) SubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToAccountBalance(h.config.AccountAddress, func(balMsg *real_time.AccountBalanceMessage) {
		select {
		case h.balanceCh <- connector.AccountBalance{
			TotalBalance:     balMsg.TotalAccountValue,
			AvailableBalance: balMsg.Withdrawable,
			UsedMargin:       balMsg.TotalMarginUsed,
			Currency:         "USD",
			UpdatedAt:        balMsg.Timestamp,
		}:
		default:
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["balance"] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeAccountBalance unsubscribes from account balance updates
func (h *hyperliquid) UnsubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	_, exists := h.subscriptions["balance"]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for balance")
	}
	delete(h.subscriptions, "balance")
	h.subMu.Unlock()

	// No unsubscribe method for account balance yet
	return nil
}

// SubscribeKlines subscribes to kline updates for an asset
func (h *hyperliquid) SubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToKlines(asset.Symbol(), interval, func(klineMsg *real_time.KlineMessage) {
		select {
		case h.klineCh <- connector.Kline{
			Symbol:    asset.Symbol(),
			Interval:  klineMsg.Interval,
			OpenTime:  klineMsg.OpenTime,
			Open:      klineMsg.Open,
			High:      klineMsg.High,
			Low:       klineMsg.Low,
			Close:     klineMsg.Close,
			Volume:    klineMsg.Volume,
			CloseTime: klineMsg.CloseTime,
		}:
		default:
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["klines:"+asset.Symbol()+":"+interval] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeKlines unsubscribes from kline updates
func (h *hyperliquid) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	h.subMu.Lock()
	key := "klines:" + asset.Symbol() + ":" + interval
	_, exists := h.subscriptions[key]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for %s", key)
	}
	delete(h.subscriptions, key)
	h.subMu.Unlock()

	return fmt.Errorf("unsubscribe klines not available in SDK")
}
