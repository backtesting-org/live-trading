package paradex

import (
	"context"
	"sync"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"

	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/connectors/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/connectors/paradex/websocket"
)

// paradex implements both Connector and WebSocketConnector interfaces
type paradex struct {
	// Existing fields
	paradexService *requests.Service
	config         *exchange.Paradex
	appLogger      logging.ApplicationLogger
	tradingLogger  logging.TradingLogger
	ctx            context.Context

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

// Ensure paradex implements both interfaces at compile time
var _ connector.Connector = (*paradex)(nil)
var _ connector.WebSocketConnector = (*paradex)(nil)

func NewParadex(
	paradexService *requests.Service,
	wsService websockets.WebSocketService,
	config *exchange.Paradex,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
) connector.Connector {
	return &paradex{
		paradexService: paradexService,
		wsService:      wsService,
		config:         config,
		appLogger:      appLogger,
		tradingLogger:  tradingLogger,
		ctx:            context.Background(),
	}
}
