package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	hyperliquidsdk "github.com/sonirico/go-hyperliquid"
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

	subID, err := h.realTime.SubscribeToOrderBook(asset.Symbol(), func(msg hyperliquidsdk.WSMessage) {
		if msg.Channel != "l2Book" {
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}

		coin, _ := data["coin"].(string)
		if coin != asset.Symbol() {
			return
		}

		levels, _ := data["levels"].([]interface{})
		if len(levels) < 2 {
			return
		}

		bids := []connector.PriceLevel{}
		asks := []connector.PriceLevel{}

		if bidData, ok := levels[0].([]interface{}); ok {
			for _, bid := range bidData {
				if bidLevel, ok := bid.(map[string]interface{}); ok {
					priceStr, _ := bidLevel["px"].(string)
					sizeStr, _ := bidLevel["sz"].(string)
					price := parseDecimal(priceStr)
					quantity := parseDecimal(sizeStr)
					bids = append(bids, connector.PriceLevel{
						Price:    price,
						Quantity: quantity,
					})
				}
			}
		}

		if askData, ok := levels[1].([]interface{}); ok {
			for _, ask := range askData {
				if askLevel, ok := ask.(map[string]interface{}); ok {
					priceStr, _ := askLevel["px"].(string)
					sizeStr, _ := askLevel["sz"].(string)
					price := parseDecimal(priceStr)
					quantity := parseDecimal(sizeStr)
					asks = append(asks, connector.PriceLevel{
						Price:    price,
						Quantity: quantity,
					})
				}
			}
		}

		orderBook := connector.OrderBook{
			Asset:     asset,
			Timestamp: time.Now(),
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

	subID, err := h.realTime.SubscribeToTrades(asset.Symbol(), func(msg hyperliquidsdk.WSMessage) {
		if msg.Channel != "trades" {
			return
		}

		var trades []interface{}
		if err := json.Unmarshal(msg.Data, &trades); err != nil {
			return
		}

		for _, tradeData := range trades {
			trade, ok := tradeData.(map[string]interface{})
			if !ok {
				continue
			}

			coin, _ := trade["coin"].(string)
			if coin != asset.Symbol() {
				continue
			}

			priceStr, _ := trade["px"].(string)
			sizeStr, _ := trade["sz"].(string)
			sideStr, _ := trade["side"].(string)
			timestamp, _ := trade["time"].(float64)

			select {
			case h.tradeCh <- connector.Trade{
				Symbol:    asset.Symbol(),
				Exchange:  connector.Hyperliquid,
				Price:     parseDecimal(priceStr),
				Quantity:  parseDecimal(sizeStr),
				Side:      connector.FromString(sideStr),
				Timestamp: time.Unix(int64(timestamp)/1000, 0),
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

	subID, err := h.realTime.SubscribeToUserEvents(h.config.AccountAddress, func(msg hyperliquidsdk.WSMessage) {
		if msg.Channel != "userEvents" {
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}

		positions, ok := data["positions"].([]interface{})
		if !ok {
			return
		}

		for _, posData := range positions {
			pos, ok := posData.(map[string]interface{})
			if !ok {
				continue
			}

			coin, _ := pos["coin"].(string)
			if coin != asset.Symbol() {
				continue
			}

			sizeStr, _ := pos["szi"].(string)
			entryPxStr, _ := pos["entryPx"].(string)
			unrealizedPnlStr, _ := pos["unrealizedPnl"].(string)

			size := parseDecimal(sizeStr)
			side := connector.OrderSideBuy
			if size.IsNegative() {
				side = connector.OrderSideSell
			}

			select {
			case h.positionCh <- connector.Position{
				Symbol:        asset,
				Exchange:      connector.Hyperliquid,
				Side:          side,
				Size:          size.Abs(),
				EntryPrice:    parseDecimal(entryPxStr),
				MarkPrice:     parseDecimal(entryPxStr),
				UnrealizedPnL: parseDecimal(unrealizedPnlStr),
				RealizedPnL:   parseDecimal("0"),
				UpdatedAt:     time.Now(),
			}:
			default:
			}
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
	subID, exists := h.subscriptions["positions:"+asset.Symbol()]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for positions:%s", asset.Symbol())
	}
	delete(h.subscriptions, "positions:"+asset.Symbol())
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromUserEvents(h.config.AccountAddress, subID)
}

// SubscribeAccountBalance subscribes to account balance updates
func (h *hyperliquid) SubscribeAccountBalance() error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToUserEvents(h.config.AccountAddress, func(msg hyperliquidsdk.WSMessage) {
		if msg.Channel != "userEvents" {
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}

		balances, ok := data["crossMarginSummary"].(map[string]interface{})
		if !ok {
			return
		}

		accountValue, _ := balances["accountValue"].(string)
		totalMarginUsed, _ := balances["totalMarginUsed"].(string)
		withdrawable, _ := balances["withdrawable"].(string)

		select {
		case h.balanceCh <- connector.AccountBalance{
			TotalBalance:     parseDecimal(accountValue),
			AvailableBalance: parseDecimal(withdrawable),
			UsedMargin:       parseDecimal(totalMarginUsed),
			Currency:         "USD",
			UpdatedAt:        time.Now(),
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
	subID, exists := h.subscriptions["balance"]
	if !exists {
		h.subMu.Unlock()
		return fmt.Errorf("no active subscription for balance")
	}
	delete(h.subscriptions, "balance")
	h.subMu.Unlock()

	return h.realTime.UnsubscribeFromUserEvents(h.config.AccountAddress, subID)
}

// SubscribeKlines subscribes to kline updates for an asset
func (h *hyperliquid) SubscribeKlines(asset portfolio.Asset, interval string) error {
	if !h.initialized {
		return fmt.Errorf("connector not initialized")
	}

	subID, err := h.realTime.SubscribeToCandles(asset.Symbol(), interval, func(msg hyperliquidsdk.WSMessage) {
		if msg.Channel != "candle" {
			return
		}

		var data map[string]interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}

		coin, _ := data["s"].(string)
		if coin != asset.Symbol() {
			return
		}

		openPrice, _ := data["o"].(string)
		highPrice, _ := data["h"].(string)
		lowPrice, _ := data["l"].(string)
		closePrice, _ := data["c"].(string)
		volume, _ := data["v"].(string)
		timestamp, _ := data["t"].(float64)

		select {
		case h.klineCh <- connector.Kline{
			Symbol:    asset.Symbol(),
			Interval:  interval,
			OpenTime:  time.Unix(int64(timestamp)/1000, 0),
			Open:      parseDecimal(openPrice),
			High:      parseDecimal(highPrice),
			Low:       parseDecimal(lowPrice),
			Close:     parseDecimal(closePrice),
			Volume:    parseDecimal(volume),
			CloseTime: time.Unix(int64(timestamp)/1000, 0),
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
