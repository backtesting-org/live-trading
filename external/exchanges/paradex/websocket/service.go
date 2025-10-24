package websockets

import (
	"context"
	"sync"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/websocket/base"
	connection2 "github.com/backtesting-org/live-trading/external/websocket/connection"
	"github.com/backtesting-org/live-trading/external/websocket/performance"
	"github.com/backtesting-org/live-trading/external/websocket/security"
)

type Service struct {
	connectionManager *connection2.ConnectionManager
	reconnectManager  *connection2.ReconnectManager
	handlerRegistry   *base.HandlerRegistry
	subManager        *subscriptionManager

	client            *adaptor.Client
	applicationLogger logging.ApplicationLogger
	tradingLogger     logging.TradingLogger

	requestID    int64
	requestMutex sync.Mutex
	writeMutex   sync.Mutex

	orderbookChan chan OrderbookUpdate
	tradeChan     chan TradeUpdate
	accountChan   chan AccountUpdate
	errorChan     chan error

	// Add kline builder
	klineBuilder *KlineBuilder
	klineChan    chan KlineUpdate
}

func NewService(
	client *adaptor.Client,
	config *exchange.Paradex,
	logger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
) *Service {
	connConfig := connection2.TradingConfig(config.WebSocketURL)
	authManager := security.NewAuthManager(&ParadexAuthProvider{client: client}, logger)
	metrics := performance.NewMetrics()
	connectionManager := connection2.NewConnectionManager(connConfig, authManager, metrics, logger)
	reconnectStrategy := connection2.NewExponentialBackoffStrategy(5*time.Second, 5*time.Minute, 10)
	reconnectManager := connection2.NewReconnectManager(connectionManager, reconnectStrategy, logger)
	handlerRegistry := base.NewHandlerRegistry(logger)

	service := &Service{
		connectionManager: connectionManager,
		reconnectManager:  reconnectManager,
		handlerRegistry:   handlerRegistry,
		subManager:        newSubscriptionManager(),
		client:            client,
		applicationLogger: logger,
		tradingLogger:     tradingLogger,

		orderbookChan: make(chan OrderbookUpdate, 1000),
		tradeChan:     make(chan TradeUpdate, 1000),
		accountChan:   make(chan AccountUpdate, 100),
		errorChan:     make(chan error, 10),

		// Initialize kline builder
		klineBuilder: NewKlineBuilder(),
		klineChan:    make(chan KlineUpdate, 1000),
	}

	service.setupCallbacks()
	service.registerHandlers()

	// Start feeding trades to kline builder
	go service.feedTradesToKlineBuilder()

	return service
}

func (s *Service) Connect(ctx context.Context) error {
	return s.connectionManager.Connect(ctx)
}

func (s *Service) Disconnect() error {
	return s.connectionManager.Disconnect()
}

func (s *Service) IsConnected() bool {
	return s.connectionManager.GetState() == connection2.StateConnected
}

func (s *Service) GetMetrics() map[string]interface{} {
	return s.connectionManager.GetConnectionStats()
}

func (s *Service) ErrorChannel() <-chan error {
	return s.errorChan
}

func (s *Service) StartWebSocket(ctx context.Context) error {
	return s.Connect(ctx)
}

func (s *Service) StopWebSocket() error {
	return s.Disconnect()
}

func (s *Service) IsWebSocketConnected() bool {
	return s.IsConnected()
}

func (s *Service) SubscribeOrderBook(asset string) error {
	return s.SubscribeOrderbook(asset)
}

func (s *Service) SubscribeTrades(asset string) error {
	return s.SubscribeTradesForSymbol(asset)
}

func (s *Service) SubscribeAccount() error {
	return s.SubscribeAccountUpdates()
}

// Thread-safe write method
func (s *Service) safeWriteJSON(message interface{}) error {
	s.writeMutex.Lock()
	defer s.writeMutex.Unlock()
	return s.connectionManager.SendJSON(message)
}

// Feed trades to kline builder
func (s *Service) feedTradesToKlineBuilder() {
	for {
		select {
		case trade, ok := <-s.tradeChan:
			if !ok {
				return
			}
			if s.klineBuilder != nil {
				s.klineBuilder.ProcessTrade(trade)
			}
		}
	}
}
