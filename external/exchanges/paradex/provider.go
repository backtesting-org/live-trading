package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/adaptor"
	"github.com/backtesting-org/live-trading/external/exchanges/paradex/requests"
	websockets "github.com/backtesting-org/live-trading/external/exchanges/paradex/websocket"
)

// NewConnector creates a new Paradex connector instance
// This is the factory function for Paradex that can be registered with the connector registry
func NewConnector(
	config *exchange.Paradex,
	appLogger logging.ApplicationLogger,
	tradingLogger logging.TradingLogger,
	timeProvider temporal.TimeProvider,
) (connector.Connector, error) {
	// Create Paradex HTTP client
	client, err := adaptor.NewClient(config, appLogger)
	if err != nil {
		return nil, err
	}

	// Create Paradex services
	requestsService := requests.NewService(client, appLogger)
	wsService := websockets.NewService(client, config, appLogger, tradingLogger, timeProvider)

	return NewParadex(requestsService, wsService, config, appLogger, tradingLogger), nil
}
