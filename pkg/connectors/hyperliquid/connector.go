package hyperliquid

import (
	"context"
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/trading"
)

// hyperliquid implements Connector and Initializable interfaces
type hyperliquid struct {
	exchangeClient clients.ExchangeClient
	infoClient     clients.InfoClient
	wsClient       clients.WebSocketClient
	marketData     data.MarketDataService
	trading        trading.TradingService
	realTime       real_time.RealTimeService
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	ctx            context.Context
	initialized    bool

	// WebSocket channels
	orderBookCh chan connector.OrderBook
	tradeCh     chan connector.Trade
	positionCh  chan connector.Position
	balanceCh   chan connector.AccountBalance
	klineCh     chan connector.Kline
	errorCh     chan error

	// Subscription tracking
	subscriptions map[string]int
	subMu         sync.RWMutex
}

// Ensure hyperliquid implements all interfaces at compile time
var _ connector.Connector = (*hyperliquid)(nil)
var _ connector.WebSocketConnector = (*hyperliquid)(nil)

// NewHyperliquid creates a new Hyperliquid connector
func NewHyperliquid(
	exchangeClient clients.ExchangeClient,
	infoClient clients.InfoClient,
	wsClient clients.WebSocketClient,
	tradingService trading.TradingService,
	marketDataService data.MarketDataService,
	realTimeService real_time.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) connector.Connector {
	return &hyperliquid{
		exchangeClient: exchangeClient,
		infoClient:     infoClient,
		wsClient:       wsClient,
		trading:        tradingService,
		marketData:     marketDataService,
		realTime:       realTimeService,
		config:         nil, // Will be set during initialization
		appLogger:      appLogger,
		tradingLogger:  tradingLogger,
		timeProvider:   timeProvider,
		ctx:            context.Background(),
		initialized:    false,
		orderBookCh:    make(chan connector.OrderBook, 100),
		tradeCh:        make(chan connector.Trade, 100),
		positionCh:     make(chan connector.Position, 100),
		balanceCh:      make(chan connector.AccountBalance, 100),
		klineCh:        make(chan connector.Kline, 100),
		errorCh:        make(chan error, 100),
		subscriptions:  make(map[string]int),
	}
}

// Initialize implements Initializable interface
func (h *hyperliquid) Initialize(config connector.Config) error {
	if h.initialized {
		return fmt.Errorf("connector already initialized")
	}

	hlConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Hyperliquid connector: expected *hyperliquid.Config, got %T", config)
	}

	// Configure the existing clients with runtime config
	if err := h.exchangeClient.Configure(hlConfig.BaseURL, hlConfig.PrivateKey, hlConfig.VaultAddress, hlConfig.AccountAddress); err != nil {
		return fmt.Errorf("failed to configure exchange client: %w", err)
	}

	if err := h.infoClient.Configure(hlConfig.BaseURL); err != nil {
		return fmt.Errorf("failed to configure info client: %w", err)
	}

	if err := h.wsClient.Configure(hlConfig.BaseURL, hlConfig.PrivateKey); err != nil {
		return fmt.Errorf("failed to configure websocket client: %w", err)
	}

	h.config = hlConfig
	h.initialized = true
	h.appLogger.Info("Hyperliquid connector initialized", "base_url", hlConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (h *hyperliquid) IsInitialized() bool {
	return h.initialized
}
