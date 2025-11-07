package websockets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func (s *Service) setupCallbacks() {
	s.connectionManager.SetCallbacks(
		s.onConnect,
		s.onDisconnect,
		s.onMessage,
		s.onError,
	)

	s.reconnectManager.SetCallbacks(
		s.onReconnectStart,
		s.onReconnectFail,
		s.onReconnectSuccess,
	)
}

func (s *Service) onConnect() error {
	s.tradingLogger.Info("Paradex WebSocket connected")
	return nil
}

func (s *Service) onDisconnect() {
	s.tradingLogger.Info("Paradex WebSocket disconnected")
}

func (s *Service) onMessage(message []byte) error {
	// Handle Paradex subscription messages directly
	if s.isParadexSubscriptionMessage(message) {
		return s.routeParadexSubscription(message)
	}

	s.applicationLogger.Debug("🔵 RECEIVED FROM PARADEX: %s", string(message))

	// Handle subscription confirmations
	if s.isSubscriptionConfirmation(message) {
		return s.handleSubscriptionConfirmation(message)
	}

	// Fallback to generic handler registry for other message types
	return s.handlerRegistry.RouteMessage(context.Background(), message)
}

func (s *Service) onError(err error) {
	select {
	case s.errorChan <- err:
	default:
	}
}

func (s *Service) onReconnectStart(attempt int) {
	s.tradingLogger.Info("Starting Paradex reconnection attempt %d", attempt)
}

func (s *Service) onReconnectFail(attempt int, err error) {
	s.tradingLogger.Info("Paradex reconnection attempt %d failed: %v", attempt, err)
}

func (s *Service) onReconnectSuccess(attempt int) {
	s.tradingLogger.Info("Paradex reconnected successfully after %d attempts", attempt)
	s.resubscribeAll()
}

func (s *Service) registerHandlers() {
	// Paradex messages are handled directly in onMessage routing
	// No need for separate handlers since we process JSON-RPC format directly
	s.applicationLogger.Debug("Paradex-specific message routing configured")
}

func (s *Service) isParadexSubscriptionMessage(message []byte) bool {
	var msg struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return false
	}

	return msg.JSONRPC == "2.0" && msg.Method == "subscription"
}

func (s *Service) routeParadexSubscription(message []byte) error {
	var msg struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Channel string          `json:"channel"`
			Data    json.RawMessage `json:"data"`
		} `json:"params"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to parse Paradex subscription: %w", err)
	}

	// Route based on channel type
	switch {
	case strings.HasPrefix(msg.Params.Channel, "order_book."):
		return s.processOrderbookData(msg.Params.Channel, msg.Params.Data)
	case strings.HasPrefix(msg.Params.Channel, "trades."):
		return s.processTradeData(msg.Params.Channel, msg.Params.Data)
	case msg.Params.Channel == "account":
		return s.processAccountData(msg.Params.Data)
	default:
		s.applicationLogger.Debug("Unknown Paradex channel: %s", msg.Params.Channel)
		return nil
	}
}

func (s *Service) isSubscriptionConfirmation(message []byte) bool {
	var msg struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int64       `json:"id"`
		Result  interface{} `json:"result,omitempty"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		return false
	}

	return msg.JSONRPC == "2.0" && msg.ID > 0 && msg.Result != nil
}

func (s *Service) handleSubscriptionConfirmation(message []byte) error {
	s.applicationLogger.Debug("📋 Subscription confirmed: %s", string(message))
	return nil
}

func (s *Service) processOrderbookData(channel string, data json.RawMessage) error {
	// Extract symbol from channel name
	symbol := s.extractSymbolFromChannel(channel)

	// Parse Paradex orderbook format
	var paradexData struct {
		SeqNo         int64  `json:"seq_no"`
		Market        string `json:"market"`
		LastUpdatedAt int64  `json:"last_updated_at"`
		UpdateType    string `json:"update_type"`
		Inserts       []struct {
			Side  string `json:"side"`
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"inserts"`
		Updates []struct {
			Side  string `json:"side"`
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"updates"`
		Deletes []struct {
			Side  string `json:"side"`
			Price string `json:"price"`
			Size  string `json:"size"`
		} `json:"deletes"`
	}

	if err := json.Unmarshal(data, &paradexData); err != nil {
		return fmt.Errorf("failed to parse Paradex orderbook data: %w", err)
	}

	// Convert to your internal format
	update := OrderbookUpdate{
		Symbol:    symbol,
		Bids:      s.convertParadexLevels(paradexData.Inserts, "BUY"),
		Asks:      s.convertParadexLevels(paradexData.Inserts, "SELL"),
		Timestamp: time.UnixMilli(paradexData.LastUpdatedAt),
		SeqNum:    paradexData.SeqNo,
	}

	// Send to orderbook channel
	select {
	case s.orderbookChan <- update:
		//s.applicationLogger.Debug("✅ Processed orderbook update for %s", symbol)
	default:
		s.applicationLogger.Warn("Orderbook channel full, dropping update for %s", symbol)
	}

	return nil
}

func (s *Service) processTradeData(channel string, data json.RawMessage) error {
	symbol := s.extractSymbolFromChannel(channel)

	var paradexTrade struct {
		ID        string `json:"id"`
		Price     string `json:"price"`
		Size      string `json:"size"`
		Side      string `json:"side"`
		Timestamp int64  `json:"created_at"`
	}

	if err := json.Unmarshal(data, &paradexTrade); err != nil {
		return fmt.Errorf("failed to parse Paradex trade data: %w", err)
	}

	// Add validation for empty fields
	if paradexTrade.Price == "" {
		s.applicationLogger.Debug("Skipping trade with empty price for %s", symbol)
		return nil
	}

	if paradexTrade.Size == "" {
		s.applicationLogger.Debug("Skipping trade with empty size for %s", symbol)
		return nil
	}

	price, err := decimal.NewFromString(paradexTrade.Price)
	if err != nil {
		return fmt.Errorf("invalid price '%s': %w", paradexTrade.Price, err)
	}

	quantity, err := decimal.NewFromString(paradexTrade.Size)
	if err != nil {
		return fmt.Errorf("invalid size '%s': %w", paradexTrade.Size, err)
	}

	update := TradeUpdate{
		Symbol:    symbol,
		Price:     price,
		Quantity:  quantity,
		Side:      paradexTrade.Side,
		Timestamp: time.UnixMilli(paradexTrade.Timestamp),
		TradeID:   paradexTrade.ID,
	}

	select {
	case s.tradeChan <- update:
	default:
		s.applicationLogger.Warn("Trade channel full, dropping update for %s", symbol)
	}

	s.applicationLogger.Debug("✅ Processed trade update for %s", symbol)
	return nil
}

func (s *Service) processAccountData(data json.RawMessage) error {
	// Parse Paradex account format
	var paradexData struct {
		UpdateType string `json:"update_type"`
		// Balance updates
		Asset     string `json:"asset,omitempty"`
		Balance   string `json:"balance,omitempty"`
		Available string `json:"available,omitempty"`
		// Position updates
		Symbol        string `json:"symbol,omitempty"`
		Size          string `json:"size,omitempty"`
		EntryPrice    string `json:"entry_price,omitempty"`
		UnrealizedPnL string `json:"unrealized_pnl,omitempty"`
		Side          string `json:"side,omitempty"`
		// Order updates
		OrderID string `json:"order_id,omitempty"`
		Status  string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(data, &paradexData); err != nil {
		return fmt.Errorf("failed to parse Paradex account data: %w", err)
	}

	var update AccountUpdate
	update.Type = paradexData.UpdateType
	update.Timestamp = time.Now()

	switch paradexData.UpdateType {
	case "balance":
		update.Symbol = paradexData.Asset
		if paradexData.Balance != "" {
			if balance, err := decimal.NewFromString(paradexData.Balance); err == nil {
				update.Balance = balance
			}
		}
		if paradexData.Available != "" {
			if available, err := decimal.NewFromString(paradexData.Available); err == nil {
				update.Available = available
			}
		}

	case "position":
		update.Symbol = paradexData.Symbol
		update.Side = paradexData.Side
		if paradexData.Size != "" {
			if size, err := decimal.NewFromString(paradexData.Size); err == nil {
				update.Size = size
			}
		}
		if paradexData.EntryPrice != "" {
			if entryPrice, err := decimal.NewFromString(paradexData.EntryPrice); err == nil {
				update.EntryPrice = entryPrice
			}
		}
		if paradexData.UnrealizedPnL != "" {
			if pnl, err := decimal.NewFromString(paradexData.UnrealizedPnL); err == nil {
				update.UnrealizedPnL = pnl
			}
		}

	case "order":
		update.Symbol = paradexData.Symbol
		update.OrderID = paradexData.OrderID
		update.Status = paradexData.Status
		update.Side = paradexData.Side
	}

	select {
	case s.accountChan <- update:
		s.applicationLogger.Debug("✅ Processed account update: %s", paradexData.UpdateType)
	default:
		s.applicationLogger.Warn("Account channel full, dropping update")
	}

	return nil
}

func (s *Service) extractSymbolFromChannel(channel string) string {
	// Extract symbol from "order_book.BTC-USD-PERP.snapshot@15@100ms@1"
	parts := strings.Split(channel, ".")
	if len(parts) >= 2 {
		return parts[1] // Returns "BTC-USD-PERP"
	}
	return "UNKNOWN"
}

func (s *Service) convertParadexLevels(levels []struct {
	Side  string `json:"side"`
	Price string `json:"price"`
	Size  string `json:"size"`
}, side string) []PriceLevel {
	var result []PriceLevel

	for _, level := range levels {
		if level.Side != side {
			continue
		}

		price, err := decimal.NewFromString(level.Price)
		if err != nil {
			continue
		}

		quantity, err := decimal.NewFromString(level.Size)
		if err != nil {
			continue
		}

		result = append(result, PriceLevel{
			Price:    price,
			Quantity: quantity,
		})
	}

	return result
}
