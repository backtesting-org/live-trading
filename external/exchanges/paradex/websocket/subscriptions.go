package websockets

import (
	"fmt"
	"strings"
	"sync"
)

type subscriptionManager struct {
	subscriptions map[string]bool
	mutex         sync.RWMutex
}

func newSubscriptionManager() *subscriptionManager {
	return &subscriptionManager{
		subscriptions: make(map[string]bool),
	}
}

func (sm *subscriptionManager) add(channel, symbol string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	key := sm.buildKey(channel, symbol)
	sm.subscriptions[key] = true
}

func (sm *subscriptionManager) remove(channel, symbol string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	key := sm.buildKey(channel, symbol)
	delete(sm.subscriptions, key)
}

func (sm *subscriptionManager) exists(channel, symbol string) bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	key := sm.buildKey(channel, symbol)
	return sm.subscriptions[key]
}

func (sm *subscriptionManager) getAll() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result := make([]string, 0, len(sm.subscriptions))
	for key := range sm.subscriptions {
		result = append(result, key)
	}
	return result
}

func (sm *subscriptionManager) buildKey(channel, symbol string) string {
	if symbol != "" {
		return fmt.Sprintf("%s_%s", channel, symbol)
	}
	return channel
}

func (s *Service) getOptimalPriceTick(symbol string) string {
	switch {
	case strings.Contains(symbol, "BTC"):
		return "1" // $1.00 for BTC
	case strings.Contains(symbol, "ETH"):
		return "1" // $1.00 for ETH (changed from 0_01)
	case strings.Contains(symbol, "SOL"):
		return "0_01" // $0.01 for SOL
	default:
		return "0_01" // Default
	}
}

func (s *Service) ensureParadexFormat(symbol string) string {
	switch symbol {
	case "BTC", "BTC-USD-PERP":
		return "BTC-USD-PERP"
	case "ETH", "ETH-USD-PERP":
		return "ETH-USD-PERP"
	case "SOL", "SOL-USD-PERP":
		return "SOL-USD-PERP"
	default:
		// If it doesn't end with -PERP, add it
		if !strings.HasSuffix(symbol, "-PERP") {
			return symbol + "-USD-PERP"
		}
		return symbol
	}
}

func (s *Service) buildOrderbookChannel(symbol string, depth int, refreshRate string, priceTick string) string {
	marketSymbol := s.ensureParadexFormat(symbol)

	return fmt.Sprintf("order_book.%s.snapshot@%d@%s@%s",
		marketSymbol, depth, refreshRate, priceTick)
}

func (s *Service) SubscribeOrderbook(symbol string) error {
	priceTick := s.getOptimalPriceTick(symbol)

	channel := s.buildOrderbookChannel(
		symbol,
		15,      // 15 levels of depth
		"100ms", // 100ms refresh rate
		priceTick,
	)

	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "subscribe",
		"params": map[string]interface{}{
			"channel": channel,
		},
	}

	s.applicationLogger.Debug("Subscribing to orderbook with channel: %s", channel)

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.add("orderbook", symbol)
	return nil
}

func (s *Service) UnsubscribeOrderbook(symbol string) error {
	if !s.subManager.exists("orderbook", symbol) {
		return nil
	}

	priceTick := s.getOptimalPriceTick(symbol)
	channel := s.buildOrderbookChannel(symbol, 15, "100ms", priceTick)

	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "unsubscribe",
		"params": map[string]interface{}{
			"channel": channel,
		},
	}

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.remove("orderbook", symbol)
	return nil
}

func (s *Service) SubscribeTradesForSymbol(symbol string) error {
	marketSymbol := s.ensureParadexFormat(symbol)
	channel := fmt.Sprintf("trades.%s", marketSymbol)

	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "subscribe",
		"params": map[string]interface{}{
			"channel": channel,
		},
	}

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.add("trades", symbol)
	return nil
}

func (s *Service) UnsubscribeTrades(symbol string) error {
	if !s.subManager.exists("trades", symbol) {
		return nil
	}

	marketSymbol := s.ensureParadexFormat(symbol)
	channel := fmt.Sprintf("trades.%s", marketSymbol)

	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "unsubscribe",
		"params": map[string]interface{}{
			"channel": channel,
		},
	}

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.remove("trades", symbol)
	return nil
}

func (s *Service) SubscribeAccountUpdates() error {
	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "subscribe",
		"params": map[string]interface{}{
			"channel": "account",
		},
	}

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.add("account", "")
	return nil
}

func (s *Service) UnsubscribeAccount() error {
	if !s.subManager.exists("account", "") {
		return nil
	}

	subMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      s.getNextRequestID(),
		"method":  "unsubscribe",
		"params": map[string]interface{}{
			"channel": "account",
		},
	}

	if err := s.safeWriteJSON(subMsg); err != nil {
		return err
	}

	s.subManager.remove("account", "")
	return nil
}

func (s *Service) GetActiveSubscriptions() []string {
	return s.subManager.getAll()
}

func (s *Service) IsSubscribed(channel, symbol string) bool {
	return s.subManager.exists(channel, symbol)
}

func (s *Service) UnsubscribeAll() error {
	subscriptions := s.GetActiveSubscriptions()

	for _, subscription := range subscriptions {
		parts := strings.Split(subscription, "_")
		channel := parts[0]

		switch channel {
		case "orderbook":
			if len(parts) > 1 {
				symbol := strings.Join(parts[1:], "_")
				if err := s.UnsubscribeOrderbook(symbol); err != nil {
					s.tradingLogger.OrderLifecycle("Failed to unsubscribe from orderbook %s: %v", symbol, err)
				}
			}
		case "trades":
			if len(parts) > 1 {
				symbol := strings.Join(parts[1:], "_")
				if err := s.UnsubscribeTrades(symbol); err != nil {
					s.tradingLogger.OrderLifecycle("Failed to unsubscribe from trades %s: %v", symbol, err)
				}
			}
		case "account":
			if err := s.UnsubscribeAccount(); err != nil {
				s.tradingLogger.OrderLifecycle("Failed to unsubscribe from account: %v", err.Error())
			}
		}
	}

	return nil
}

func (s *Service) resubscribeAll() {
	subscriptions := s.GetActiveSubscriptions()

	for _, subscription := range subscriptions {
		parts := strings.Split(subscription, "_")
		channel := parts[0]

		switch channel {
		case "orderbook":
			if len(parts) > 1 {
				symbol := strings.Join(parts[1:], "_")
				if err := s.SubscribeOrderbook(symbol); err != nil {
					s.tradingLogger.OrderLifecycle("Failed to resubscribe to orderbook %s: %v", symbol, err)
				}
			}
		case "trades":
			if len(parts) > 1 {
				symbol := strings.Join(parts[1:], "_")
				if err := s.SubscribeTradesForSymbol(symbol); err != nil {
					s.tradingLogger.OrderLifecycle("Failed to resubscribe to trades %s: %v", symbol, err)
				}
			}
		case "account":
			if err := s.SubscribeAccountUpdates(); err != nil {
				s.tradingLogger.OrderLifecycle("Failed to resubscribe to account: %v", err.Error())
			}
		}
	}
}

func (s *Service) getNextRequestID() int64 {
	s.requestMutex.Lock()
	defer s.requestMutex.Unlock()
	s.requestID++
	return s.requestID
}
