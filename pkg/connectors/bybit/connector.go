package bybit

import (
	"context"
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit/trading"

	liveconnector "github.com/backtesting-org/live-trading/pkg/connector"
)

type bybit struct {
	marketData    data.MarketDataService
	trading       trading.TradingService
	realTime      real_time.RealTimeService
	config        *Config
	appLogger     logging.ApplicationLogger
	tradingLogger logging.TradingLogger
	timeProvider  temporal.TimeProvider
	ctx           context.Context
	initialized   bool

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

var _ connector.Connector = (*bybit)(nil)
var _ connector.WebSocketConnector = (*bybit)(nil)
var _ liveconnector.Initializable = (*bybit)(nil)

func NewBybit(
	tradingService trading.TradingService,
	marketDataService data.MarketDataService,
	realTimeService real_time.RealTimeService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) liveconnector.Initializable {
	return &bybit{
		trading:       tradingService,
		marketData:    marketDataService,
		realTime:      realTimeService,
		config:        nil,
		appLogger:     appLogger,
		tradingLogger: tradingLogger,
		timeProvider:  timeProvider,
		ctx:           context.Background(),
		initialized:   false,
		orderBookCh:   make(chan connector.OrderBook, 100),
		tradeCh:       make(chan connector.Trade, 100),
		positionCh:    make(chan connector.Position, 100),
		balanceCh:     make(chan connector.AccountBalance, 100),
		klineCh:       make(chan connector.Kline, 100),
		errorCh:       make(chan error, 100),
		subscriptions: make(map[string]int),
	}
}

func (b *bybit) Initialize(config liveconnector.Config) error {
	if b.initialized {
		return fmt.Errorf("connector already initialized")
	}

	bybitConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Bybit connector: expected *bybit.Config, got %T", config)
	}

	tradingConfig := &trading.Config{
		APIKey:          bybitConfig.APIKey,
		APISecret:       bybitConfig.APISecret,
		BaseURL:         bybitConfig.BaseURL,
		IsTestnet:       bybitConfig.IsTestnet,
		DefaultSlippage: bybitConfig.DefaultSlippage,
	}

	dataConfig := &data.Config{
		APIKey:          bybitConfig.APIKey,
		APISecret:       bybitConfig.APISecret,
		BaseURL:         bybitConfig.BaseURL,
		IsTestnet:       bybitConfig.IsTestnet,
		DefaultSlippage: bybitConfig.DefaultSlippage,
	}

	realTimeConfig := &real_time.Config{
		APIKey:    bybitConfig.APIKey,
		APISecret: bybitConfig.APISecret,
		BaseURL:   bybitConfig.BaseURL,
	}

	if err := b.trading.Initialize(tradingConfig); err != nil {
		return fmt.Errorf("failed to initialize trading service: %w", err)
	}

	if err := b.marketData.Initialize(dataConfig); err != nil {
		return fmt.Errorf("failed to initialize market data service: %w", err)
	}

	if err := b.realTime.Initialize(realTimeConfig); err != nil {
		return fmt.Errorf("failed to initialize real-time service: %w", err)
	}

	b.config = bybitConfig
	b.initialized = true
	b.appLogger.Info("Bybit connector initialized", "testnet", bybitConfig.IsTestnet)
	return nil
}

// Reset implements connector.Connector interface
// For live exchanges, reset is a no-op since they don't maintain simulated state
func (b *bybit) Reset() error {
	return nil
}

// IsInitialized implements Initializable interface
func (b *bybit) IsInitialized() bool {
	return b.initialized
}

func (b *bybit) Name() string {
	return "Bybit"
}

func (b *bybit) SupportedInstruments() []connector.Instrument {
	return []connector.Instrument{
		connector.TypePerpetual,
		connector.TypeSpot,
	}
}

func (b *bybit) SupportsMarketData() bool {
	return true
}

func (b *bybit) GetPerpSymbol(asset portfolio.Asset) string {
	return asset.Symbol() + "USDT"
}
