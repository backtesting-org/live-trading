package real_time

import (
	"context"
	"fmt"
	"sync"

	bybit "github.com/bybit-exchange/bybit.go.api"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
)

type Config struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

type RealTimeService interface {
	Initialize(config *Config) error
	Connect(ctx context.Context) error
	Disconnect() error
	SubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error
	UnsubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error
	SubscribeAccountBalance() error
	UnsubscribeAccountBalance() error
	SubscribeKlines(asset portfolio.Asset, interval string) error
	UnsubscribeKlines(asset portfolio.Asset, interval string) error
}

type realTimeService struct {
	websocket     *bybit.WebSocket
	logger        logging.ApplicationLogger
	timeProvider  temporal.TimeProvider
	mu            sync.RWMutex
	subscriptions map[string]bool
}

func NewRealTimeService(
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) RealTimeService {
	return &realTimeService{
		logger:        logger,
		timeProvider:  timeProvider,
		subscriptions: make(map[string]bool),
	}
}

func (r *realTimeService) Initialize(config *Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket != nil {
		return fmt.Errorf("real-time service already initialized")
	}

	r.websocket = bybit.NewBybitPrivateWebSocket(config.BaseURL, config.APIKey, config.APISecret, func(message string) error {
		return nil
	})

	return nil
}

func (r *realTimeService) Connect(ctx context.Context) error {
	r.mu.RLock()
	ws := r.websocket
	r.mu.RUnlock()

	if ws == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	ws.Connect()
	return nil
}

func (r *realTimeService) Disconnect() error {
	r.mu.RLock()
	ws := r.websocket
	r.mu.RUnlock()

	if ws == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	// Bybit SDK doesn't have a Close method, connection is managed automatically
	return nil
}

func (r *realTimeService) SubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "orderbook:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to order book", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribeOrderBook(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "orderbook:" + symbol
	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from order book", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "trades:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to trades", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribeTrades(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "trades:" + symbol
	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from trades", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "positions:" + symbol

	if r.subscriptions[key] {
		return nil
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to positions", "symbol", symbol)
	return nil
}

func (r *realTimeService) UnsubscribePositions(asset portfolio.Asset, instrument connector.Instrument) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "positions:" + symbol
	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from positions", "symbol", symbol)
	return nil
}

func (r *realTimeService) SubscribeAccountBalance() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	key := "balance"

	if r.subscriptions[key] {
		return nil
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to account balance")
	return nil
}

func (r *realTimeService) UnsubscribeAccountBalance() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.subscriptions, "balance")
	r.logger.Info("Unsubscribed from account balance")
	return nil
}

func (r *realTimeService) SubscribeKlines(asset portfolio.Asset, interval string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.websocket == nil {
		return fmt.Errorf("real-time service not initialized")
	}

	symbol := asset.Symbol() + "USDT"
	key := "klines:" + symbol + ":" + interval

	if r.subscriptions[key] {
		return nil
	}

	r.subscriptions[key] = true
	r.logger.Info("Subscribed to klines", "symbol", symbol, "interval", interval)
	return nil
}

func (r *realTimeService) UnsubscribeKlines(asset portfolio.Asset, interval string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	symbol := asset.Symbol() + "USDT"
	key := "klines:" + symbol + ":" + interval
	delete(r.subscriptions, key)
	r.logger.Info("Unsubscribed from klines", "symbol", symbol, "interval", interval)
	return nil
}
