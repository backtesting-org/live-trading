package paradex

import (
	"context"
	"fmt"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"

	"github.com/backtesting-org/live-trading/external/connectors/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/connectors/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/connectors/paradex/websocket"
	liveconnector "github.com/backtesting-org/live-trading/pkg/connector"
)

// paradex implements Connector, WebSocketConnector, and Initializable interfaces
type paradex struct {
	paradexService *requests.Service
	config         *Config
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	timeProvider   temporal.TimeProvider
	ctx            context.Context
	initialized    bool

	// WebSocket service
	wsService websockets.WebSocketService

	// WebSocket state management
	wsContext context.Context
	wsCancel  context.CancelFunc
	wsMutex   sync.RWMutex
}

// Reset implements connector.Connector interface
// For live exchanges, reset is a no-op since they don't maintain simulated state
func (p *paradex) Reset() error {
	// Live exchanges don't maintain internal simulation state to reset
	return nil
}

// Ensure paradex implements all interfaces at compile time
var _ connector.Connector = (*paradex)(nil)
var _ connector.WebSocketConnector = (*paradex)(nil)
var _ liveconnector.Initializable = (*paradex)(nil)

func NewParadex(
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) liveconnector.Initializable {
	return &paradex{
		paradexService: nil, // Will be created during initialization
		wsService:      nil, // Will be created during initialization
		config:         nil, // Will be set during initialization
		appLogger:      appLogger,
		tradingLogger:  tradingLogger,
		timeProvider:   timeProvider,
		ctx:            context.Background(),
		initialized:    false,
	}
}

// Initialize implements Initializable interface
func (p *paradex) Initialize(config liveconnector.Config) error {
	if p.initialized {
		return fmt.Errorf("connector already initialized")
	}

	paradexConfig, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for Paradex connector: expected *paradex.Config, got %T", config)
	}

	// Create adaptor client
	adaptorConfig := &adaptor.Config{
		BaseURL:       paradexConfig.BaseURL,
		StarknetRPC:   paradexConfig.StarknetRPC,
		EthPrivateKey: paradexConfig.EthPrivateKey,
		Network:       paradexConfig.Network,
	}

	client, err := adaptor.NewClient(adaptorConfig, p.appLogger)
	if err != nil {
		return fmt.Errorf("failed to create Paradex client: %w", err)
	}

	// Create services
	p.paradexService = requests.NewService(client, p.appLogger)
	p.wsService = websockets.NewService(client, paradexConfig.WebSocketURL, p.appLogger, p.tradingLogger, p.timeProvider)

	p.config = paradexConfig
	p.initialized = true
	p.appLogger.Info("Paradex connector initialized", "base_url", paradexConfig.BaseURL)
	return nil
}

// IsInitialized implements Initializable interface
func (p *paradex) IsInitialized() bool {
	return p.initialized
}
