package hyperliquid

import (
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/clients"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/data/real_time"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid/trading"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"github.com/backtesting-org/live-trading/pkg/websocket/base"
	"github.com/backtesting-org/live-trading/pkg/websocket/performance"
	"github.com/backtesting-org/live-trading/pkg/websocket/security"
	"go.uber.org/fx"
)

// WebSocket Module - Factory functions for Hyperliquid-specific dependencies

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
			real_time.NewRealTimeService,
			fx.ParamTags(`name:"hyperliquid_base"`),
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
