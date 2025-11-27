package runtime

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
)

type Runtime interface {
	Execute(strategyPath string, connectors map[connector.ExchangeName]pkgconnector.Config) error
}

func NewRuntime(
	registry registry.ConnectorRegistry,
	pluginManager plugin.Manager,
	logger logging.ApplicationLogger,
) Runtime {
	return &runtime{
		registry:      registry,
		pluginManager: pluginManager,
		logger:        logger,
	}
}

type runtime struct {
	registry      registry.ConnectorRegistry
	pluginManager plugin.Manager
	logger        logging.ApplicationLogger
}

// Execute runs the trading strategy
func (r *runtime) Execute(
	strategyPath string,
	connectors map[connector.ExchangeName]pkgconnector.Config,
) error {
	strat, err := r.pluginManager.LoadStrategyPlugin(strategyPath)
	if err != nil {
		r.logger.Error(fmt.Sprintf("Failed to load plugin: %s", err))
		return err
	}

	r.logger.Info("Successfully loaded strategy plugin", "strategy", strat.GetName())
	r.logger.Info("Starting strategy execution", "strategy", strat.GetName())

	for name, config := range connectors {
		conn, exists := r.registry.GetConnector(name)

		if !exists {
			return fmt.Errorf("failed to get connector for exchange %s", name)
		}

		r.logger.Info("Connecting to exchange", "exchange", name)

		// Initialize connector with config
		initConn, ok := conn.(pkgconnector.Initializable)
		if !ok {
			r.logger.Error(fmt.Sprintf("Connector %s does not support initialization", name))
			return fmt.Errorf("connector %s does not support initialization", name)
		}

		if err := initConn.Initialize(config); err != nil {
			return fmt.Errorf("failed to initialize connector: %w", err)
		}

		r.logger.Info("Successfully connected to exchange", "exchange", name)
	}

	return nil
}
