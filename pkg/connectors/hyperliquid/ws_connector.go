package hyperliquid

import (
	"context"
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
)

// StartWebSocket starts the WebSocket connection for real-time data
func (h *hyperliquid) StartWebSocket(ctx context.Context) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	// Start error forwarding from realTime service
	go h.forwardWebSocketErrors()

	return h.realTime.Connect()
}

// forwardWebSocketErrors forwards errors from the realTime service to the connector's error channel
func (h *hyperliquid) forwardWebSocketErrors() {
	errCh := h.realTime.GetErrorChannel()
	for err := range errCh {
		select {
		case h.errorCh <- err:
		default:
			h.appLogger.Warn("Error channel full, dropping websocket error: %v", err)
		}
	}
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
	return h.initialized && h.realTime != nil
}

// OrderBookUpdates returns a channel for order book updates
// DEPRECATED: Use GetAllOrderBookChannels() instead for proper per-asset routing
func (h *hyperliquid) OrderBookUpdates() <-chan connector.OrderBook {
	h.appLogger.Warn("⚠️  OrderBookUpdates() is deprecated - use GetAllOrderBookChannels() for proper routing")
	// Return nil to force migration to new method
	return nil
}

// GetOrderBookChannel returns the channel for a specific asset subscription
func (h *hyperliquid) GetOrderBookChannel(asset portfolio.Asset) <-chan connector.OrderBook {
	symbol := h.normaliseAssetName(asset)

	h.orderBookMu.RLock()
	defer h.orderBookMu.RUnlock()

	ch, exists := h.orderBookChannels[symbol]
	if !exists {
		h.appLogger.Error("❌ No orderbook channel found for %s", symbol)
		return nil
	}

	h.appLogger.Info("📊 Returning orderbook channel %p for %s", ch, symbol)
	return ch
}

// GetAllOrderBookChannels returns all active orderbook channels (for ingestor to read from all)
func (h *hyperliquid) GetAllOrderBookChannels() map[string]<-chan connector.OrderBook {
	h.orderBookMu.RLock()
	defer h.orderBookMu.RUnlock()

	result := make(map[string]<-chan connector.OrderBook, len(h.orderBookChannels))
	for key, ch := range h.orderBookChannels {
		result[key] = ch
	}

	h.appLogger.Info("📊 Returning %d orderbook channels", len(result))
	return result
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
	// This method is deprecated - use GetKlineChannel instead
	// Return nil to force users to use the new method
	h.appLogger.Warn("⚠️  KlineUpdates() is deprecated - use connector-specific channel access")
	return nil
}

// GetKlineChannel returns the channel for a specific asset/interval subscription
func (h *hyperliquid) GetKlineChannel(asset portfolio.Asset, interval string) <-chan connector.Kline {
	symbol := h.normaliseAssetName(asset)
	channelKey := fmt.Sprintf("%s:%s", symbol, interval)

	h.klineMu.RLock()
	defer h.klineMu.RUnlock()

	ch, exists := h.klineChannels[channelKey]
	if !exists {
		h.appLogger.Error("❌ No kline channel found for %s", channelKey)
		return nil
	}

	h.appLogger.Info("📊 Returning kline channel %p for %s", ch, channelKey)
	return ch
}

// GetAllKlineChannels returns all active kline channels (for ingestor to read from all)
func (h *hyperliquid) GetAllKlineChannels() map[string]<-chan connector.Kline {
	h.klineMu.RLock()
	defer h.klineMu.RUnlock()

	result := make(map[string]<-chan connector.Kline, len(h.klineChannels))
	for key, ch := range h.klineChannels {
		result[key] = ch
	}

	h.appLogger.Info("📊 Returning %d kline channels", len(result))
	return result
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

	symbol := h.normaliseAssetName(asset)

	// Create dedicated channel for this asset if it doesn't exist
	h.orderBookMu.Lock()
	orderBookCh, exists := h.orderBookChannels[symbol]
	if !exists {
		orderBookCh = make(chan connector.OrderBook, 100)
		h.orderBookChannels[symbol] = orderBookCh
		h.appLogger.Info("🔗 Created NEW orderbook channel for %s: %p", symbol, orderBookCh)
	} else {
		h.appLogger.Info("🔗 Reusing EXISTING orderbook channel for %s: %p", symbol, orderBookCh)
	}
	h.orderBookMu.Unlock()

	subID, err := h.realTime.SubscribeToOrderBook(symbol, func(obMsg *real_time.OrderBookMessage) {
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
		case orderBookCh <- orderBook:
			h.appLogger.Debug("📊 Sent orderbook for %s to dedicated channel %p (bids: %d, asks: %d)", symbol, orderBookCh, len(bids), len(asks))
		default:
			h.appLogger.Warn("⚠️  Orderbook channel FULL for %s - dropping update (channel buffer: 100)", symbol)
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["orderbook:"+symbol] = subID
	h.subMu.Unlock()

	h.appLogger.Info("✅ Subscribed to orderbook for %s (subID: %d)", symbol, subID)
	return nil
}

// UnsubscribeOrderBook unsubscribes from order book updates
func (h *hyperliquid) UnsubscribeOrderBook(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	subID, exists := h.subscriptions["orderbook:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for orderbook:%s", symbol)
	}
	delete(h.subscriptions, "orderbook:"+symbol)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromOrderBook(symbol, subID)
}

// SubscribeTrades subscribes to trade updates for an asset
func (h *hyperliquid) SubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	subID, err := h.realTime.SubscribeToTrades(symbol, func(trades []real_time.TradeMessage) {
		for _, trade := range trades {
			select {
			case h.tradeCh <- connector.Trade{
				Symbol:    trade.Coin,
				Exchange:  types.Hyperliquid,
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
	h.subscriptions["trades:"+symbol] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeTrades unsubscribes from trade updates
func (h *hyperliquid) UnsubscribeTrades(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	subID, exists := h.subscriptions["trades:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for trades:%s", symbol)
	}
	delete(h.subscriptions, "trades:"+symbol)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromTrades(symbol, subID)
}

// SubscribePositions subscribes to position updates
func (h *hyperliquid) SubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	subID, err := h.realTime.SubscribeToPositions(h.config.AccountAddress, func(posMsg *real_time.PositionMessage) {
		if posMsg.Coin != symbol {
			return
		}

		side := connector.OrderSideBuy
		if posMsg.Size.IsNegative() {
			side = connector.OrderSideSell
		}

		select {
		case h.positionCh <- connector.Position{
			Symbol:        asset,
			Exchange:      types.Hyperliquid,
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
	h.subscriptions["positions:"+symbol] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribePositions unsubscribes from position updates
func (h *hyperliquid) UnsubscribePositions(asset portfolio.Asset, instrumentType connector.Instrument) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	_, exists := h.subscriptions["positions:"+symbol]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for positions:%s", symbol)
	}
	delete(h.subscriptions, "positions:"+symbol)
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

	symbol := h.normaliseAssetName(asset)
	channelKey := fmt.Sprintf("%s:%s", symbol, interval)

	h.appLogger.Info("🔗 SubscribeKlines called on connector %p for %s", h, channelKey)

	// Create dedicated channel for this subscription
	h.klineMu.Lock()
	klineCh := make(chan connector.Kline, 100)
	h.klineChannels[channelKey] = klineCh
	h.klineMu.Unlock()

	subID, err := h.realTime.SubscribeToKlines(symbol, interval, func(klineMsg *real_time.KlineMessage) {
		// CRITICAL: Only process klines matching the subscribed interval
		// Hyperliquid sends ALL intervals even if you only subscribe to one
		if klineMsg.Interval != interval {
			h.appLogger.Debug("⏭️  Skipping %s kline (subscribed to %s)", klineMsg.Interval, interval)
			return
		}

		kline := connector.Kline{
			Symbol:    symbol,
			Interval:  klineMsg.Interval,
			OpenTime:  klineMsg.OpenTime,
			Open:      klineMsg.Open,
			High:      klineMsg.High,
			Low:       klineMsg.Low,
			Close:     klineMsg.Close,
			Volume:    klineMsg.Volume,
			CloseTime: klineMsg.CloseTime,
		}

		select {
		case klineCh <- kline:
			h.appLogger.Debug("✅ Sent %s %s kline to dedicated channel", symbol, interval)
		default:
			h.appLogger.Warn("⚠️  Channel full for %s - dropping update (buffer: 100)", channelKey)
		}
	})
	if err != nil {
		return err
	}

	h.subMu.Lock()
	h.subscriptions["klines:"+symbol+":"+interval] = subID
	h.subMu.Unlock()
	return nil
}

// UnsubscribeKlines unsubscribes from kline updates
func (h *hyperliquid) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	symbol := h.normaliseAssetName(asset)

	h.subMu.Lock()
	key := "klines:" + symbol + ":" + interval
	subID, exists := h.subscriptions[key]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for %s", key)
	}
	delete(h.subscriptions, key)
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromKlines(symbol, interval, subID)
}
