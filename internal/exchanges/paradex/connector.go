package paradex

import (
	"context"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/exchanges/paradex/websocket"
)

// Paradex implements both Connector and WebSocketConnector interfaces
type Paradex struct {
	// Existing fields
	paradexService *requests.Service
	config         *exchange.Paradex
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	ctx            context.Context

	// WebSocket service
	wsService *websockets.Service

	// WebSocket state management
	wsContext context.Context
	wsCancel  context.CancelFunc
	wsMutex   sync.RWMutex
}

// Reset implements connector.Connector interface
// For live exchanges, reset is a no-op since they don't maintain simulated state
func (p *Paradex) Reset() error {
	// Live exchanges don't maintain internal simulation state to reset
	return nil
}

// Ensure Paradex implements both interfaces at compile time
var _ connector.Connector = (*Paradex)(nil)
var _ connector.WebSocketConnector = (*Paradex)(nil)

func NewParadex(
	paradexService *requests.Service,
	wsService *websockets.Service,
	config *exchange.Paradex,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
) *Paradex {
	return &Paradex{
		paradexService: paradexService,
		wsService:      wsService,
		config:         config,
		appLogger:      appLogger,
		tradingLogger:  tradingLogger,
		ctx:            context.Background(),
	}
}
