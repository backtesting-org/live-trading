package hyperliquid

import (
	"context"
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"

	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid/trading"
	liveconnector "github.com/backtesting-org/live-trading/pkg/connector"
)

// hyperliquid implements Connector and Initializable interfaces
type hyperliquid struct {
	exchangeClient clients.ExchangeClient
	infoClient     clients.InfoClient
	marketData     data.MarketDataService
	trading        trading.TradingService
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	ctx            context.Context
	initialized    bool
}

// Ensure hyperliquid implements all interfaces at compile time
var _ connector.Connector = (*hyperliquid)(nil)
var _ connector.WebSocketConnector = (*hyperliquid)(nil)
var _ liveconnector.Initializable = (*hyperliquid)(nil)

// NewHyperliquid creates a new Hyperliquid connector (not yet initialized)
func NewHyperliquid(
	exchangeClient clients.ExchangeClient,
	infoClient clients.InfoClient,
	tradingService trading.TradingService,
	marketDataService data.MarketDataService,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) liveconnector.Initializable {
	return &hyperliquid{
		exchangeClient: exchangeClient,
		infoClient:     infoClient,
		trading:        tradingService,
		marketData:     marketDataService,
		config:         nil, // Will be set during initialization
		appLogger:      appLogger,
		tradingLogger:  tradingLogger,
		timeProvider:   timeProvider,
		ctx:            context.Background(),
		initialized:    false,
	}
}

// Initialize implements Initializable interface
func (h *hyperliquid) Initialize(config liveconnector.Config) error {
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

	h.config = hlConfig
	h.initialized = true
	h.appLogger.Info("Hyperliquid connector initialized", "base_url", hlConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (h *hyperliquid) IsInitialized() bool {
	return h.initialized
}

// Reset implements connector.Connector interface
// For live exchanges, reset is a no-op since they don't maintain simulated state
func (h *hyperliquid) Reset() error {
	// Live exchanges don't maintain internal simulation state to reset
	return nil
}
