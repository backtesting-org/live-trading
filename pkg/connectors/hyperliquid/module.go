package hyperliquid

import (
	"context"
	"net/http"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/kronos-sdk/pkg/types/temporal"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/trading"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
	"go.uber.org/fx"
)

// realTimeServiceFactory creates a RealTimeService using pkg/websocket infrastructure
func realTimeServiceFactory(
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) real_time.RealTimeService {
	// Default WebSocket URL - will be overridden during connector initialization
	wsURL := "wss://api.hyperliquid.xyz/ws"

	// Create a no-op auth provider for Hyperliquid (it doesn't require auth for public channels)
	authProvider := &noOpAuthProvider{}

	// Create auth manager with the no-op provider
	authManager := security.NewAuthManager(authProvider, logger)

	// Create the RealTimeService with the new implementation
	svc, err := real_time.NewRealTimeService(wsURL, authManager, logger, timeProvider)
	if err != nil {
		logger.Error("Failed to create RealTimeService: %v", err)
		panic(err)
	}
	return svc
}

// noOpAuthProvider is a no-op implementation for public WebSocket channels
type noOpAuthProvider struct{}

func (n *noOpAuthProvider) GetAuthHeaders(_ context.Context) (http.Header, error) {
	return make(http.Header), nil
}

func (n *noOpAuthProvider) IsAuthenticated() bool {
	return true
}

func (n *noOpAuthProvider) Refresh(_ context.Context) error {
	return nil
}

func (n *noOpAuthProvider) GetTokenExpiry() time.Time {
	return time.Now().Add(24 * time.Hour)
}

var Module = fx.Options(
	fx.Provide(
		clients.NewExchangeClient,
		clients.NewInfoClient,
		clients.NewWebSocketClient,
		trading.NewTradingService,
		data.NewMarketDataService,
		realTimeServiceFactory,
		fx.Annotate(
			NewHyperliquid,
			fx.ResultTags(`name:"hyperliquid"`),
		),
	),
	// Automatically register hyperliquid with the SDK registry at startup
	fx.Invoke(fx.Annotate(
		registerHyperliquid,
		fx.ParamTags(`name:"hyperliquid"`),
	)),
)

// registerHyperliquid registers the hyperliquid connector with the SDK's ConnectorRegistry
func registerHyperliquid(hyperliquidConn connector.Connector, reg registry.ConnectorRegistry) {
	// Register the connector (Initializable embeds connector.Connector)
	reg.RegisterConnector(types.Hyperliquid, hyperliquidConn)
}
