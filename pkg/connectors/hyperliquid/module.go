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
	"github.com/backtesting-org/live-trading/pkg/websocket/base"
	"github.com/backtesting-org/live-trading/pkg/websocket/connection"
	"github.com/backtesting-org/live-trading/pkg/websocket/performance"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
	"go.uber.org/fx"
)

// WebSocket Module - Factory functions for Hyperliquid-specific dependencies

// NewHyperliquidAuthManager creates auth manager for Hyperliquid (no-op auth)
func NewHyperliquidAuthManager(logger logging.ApplicationLogger) security.AuthManager {
	// Hyperliquid public streams don't require authentication
	authProvider := &noOpAuthProvider{}
	return security.NewAuthManager(authProvider, logger)
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

// NewHyperliquidValidationConfig creates Hyperliquid's specific validation config
func NewHyperliquidValidationConfig() security.ValidationConfig {
	return DefaultWSConfig().ValidationConfig()
}

// NewHyperliquidMessageValidator creates validator with Hyperliquid config
func NewHyperliquidMessageValidator(valConfig security.ValidationConfig) security.MessageValidator {
	return security.NewMessageValidator(valConfig)
}

// NewHyperliquidRateLimiter creates rate limiter with Hyperliquid settings
func NewHyperliquidRateLimiter(cfg HyperliquidWSConfig) security.RateLimiter {
	return security.NewRateLimiter(cfg.RateLimitCapacity, cfg.RateLimitRefill)
}

// NewHyperliquidMetrics creates metrics instance for Hyperliquid
func NewHyperliquidMetrics() performance.Metrics {
	return performance.NewMetrics()
}

// NewHyperliquidCircuitBreaker creates circuit breaker with Hyperliquid settings
func NewHyperliquidCircuitBreaker() performance.CircuitBreaker {
	return performance.NewCircuitBreaker(3, 30*time.Second)
}

// NewHyperliquidParser creates a message parser for Hyperliquid WebSocket messages
func NewHyperliquidParser(
	logger logging.ApplicationLogger,
	timeProvider temporal.TimeProvider,
) real_time.MessageParser {
	return real_time.NewParser(logger, timeProvider)
}

// NewHyperliquidConnectionManager creates connection manager with Hyperliquid settings
func NewHyperliquidConnectionManager(
	metrics performance.Metrics,
	authManager security.AuthManager,
	logger logging.ApplicationLogger,
) connection.ConnectionManager {
	cfg := DefaultWSConfig()
	connConfig := connection.DefaultConfig()
	connConfig.URL = cfg.WSURL
	connConfig.EnableHealthMonitoring = true
	connConfig.EnableHealthPings = true
	connConfig.HealthCheckInterval = 30 * time.Second

	return connection.NewConnectionManager(connConfig, authManager, metrics, logger)
}

// NewHyperliquidReconnectManager creates reconnect manager with Hyperliquid strategy
func NewHyperliquidReconnectManager(
	connManager connection.ConnectionManager,
	logger logging.ApplicationLogger,
) connection.ReconnectManager {
	strategy := connection.NewExponentialBackoffStrategy(
		5*time.Second,  // Initial delay
		60*time.Second, // Max delay
		10,             // Max attempts
	)
	return connection.NewReconnectManager(connManager, strategy, logger)
}

// NewHyperliquidBaseService wires all Hyperliquid WebSocket dependencies
func NewHyperliquidBaseService(
	logger logging.ApplicationLogger,
	validator security.MessageValidator,
	rateLimiter security.RateLimiter,
	metrics performance.Metrics,
	circuitBreaker performance.CircuitBreaker,
) base.BaseService {
	cfg := DefaultWSConfig()
	baseConfig := base.Config{
		URL:            cfg.WSURL,
		ReconnectDelay: cfg.ReconnectDelay,
		MaxReconnects:  cfg.MaxReconnects,
		PingInterval:   cfg.PingInterval,
		PongTimeout:    cfg.PongTimeout,
		MaxMessageSize: cfg.MaxMessageSize,
	}

	return base.NewBaseService(
		baseConfig,
		logger,
		validator,
		rateLimiter,
		metrics,
		circuitBreaker,
	)
}

// WebSocket Module registration
var WebSocketModule = fx.Module("hyperliquid_websocket",
	fx.Provide(
		fx.Annotate(
			NewHyperliquidAuthManager,
			fx.ResultTags(`name:"hyperliquid_auth_manager"`),
		),
		fx.Annotate(
			NewHyperliquidValidationConfig,
			fx.ResultTags(`name:"hyperliquid_validation"`),
		),
		fx.Annotate(
			NewHyperliquidMessageValidator,
			fx.ParamTags(`name:"hyperliquid_validation"`),
			fx.ResultTags(`name:"hyperliquid_validator"`),
		),
		fx.Annotate(
			NewHyperliquidRateLimiter,
			fx.ResultTags(`name:"hyperliquid_rate_limiter"`),
		),
		fx.Annotate(
			NewHyperliquidMetrics,
			fx.ResultTags(`name:"hyperliquid_metrics"`),
		),
		fx.Annotate(
			NewHyperliquidCircuitBreaker,
			fx.ResultTags(`name:"hyperliquid_circuit_breaker"`),
		),
		fx.Annotate(
			NewHyperliquidParser,
			fx.ResultTags(`name:"hyperliquid_parser"`),
		),
		fx.Annotate(
			NewHyperliquidConnectionManager,
			fx.ParamTags(
				`name:"hyperliquid_metrics"`,
				`name:"hyperliquid_auth_manager"`,
			),
			fx.ResultTags(`name:"hyperliquid_connection_manager"`),
		),
		fx.Annotate(
			NewHyperliquidReconnectManager,
			fx.ParamTags(
				`name:"hyperliquid_connection_manager"`,
			),
			fx.ResultTags(`name:"hyperliquid_reconnect_manager"`),
		),
		fx.Annotate(
			real_time.NewWebSocketService,
			fx.ParamTags(
				`name:"hyperliquid_connection_manager"`,
				`name:"hyperliquid_reconnect_manager"`,
				`name:"hyperliquid_base"`,
			),
			fx.ResultTags(`name:"hyperliquid_websocket"`),
		),
		fx.Annotate(
			NewHyperliquidBaseService,
			fx.ParamTags(
				`name:"hyperliquid_validator"`,
				`name:"hyperliquid_rate_limiter"`,
				`name:"hyperliquid_metrics"`,
				`name:"hyperliquid_circuit_breaker"`,
			),
			fx.ResultTags(`name:"hyperliquid_base"`),
		),
	),
)

// Exchange Connector Module - Original connector registration

var Module = fx.Options(
	WebSocketModule,
	fx.Provide(
		clients.NewExchangeClient,
		clients.NewInfoClient,
		clients.NewWebSocketClient,
		trading.NewTradingService,
		data.NewMarketDataService,
		fx.Annotate(
			real_time.NewWebSocketService,
			fx.ParamTags(
				`name:"hyperliquid_connection_manager"`,
				`name:"hyperliquid_reconnect_manager"`,
				`name:"hyperliquid_base"`,
				``,
				`name:"hyperliquid_parser"`,
			),
		),
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
