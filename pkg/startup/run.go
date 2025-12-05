package startup

import (
	"context"
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/kronos-sdk/pkg/types/runtime"
)

type Startup interface {
	Start(
		strategyPath string,
		connectors map[connector.ExchangeName]connector.Config,
		assets map[portfolio.Asset][]connector.Instrument,
	) error
	Stop() error
}

func NewStartup(
	connectorRegistry registry.ConnectorRegistry,
	assetRegistry registry.AssetRegistry,
	pluginManager plugin.Manager,
	runtime runtime.Runtime,
	logger logging.ApplicationLogger,
) Startup {
	return &startup{
		connectorRegistry: connectorRegistry,
		assetRegistry:     assetRegistry,
		runtime:           runtime,
		pluginManager:     pluginManager,
		logger:            logger,
	}
}

type startup struct {
	connectorRegistry registry.ConnectorRegistry
	assetRegistry     registry.AssetRegistry
	pluginManager     plugin.Manager
	runtime           runtime.Runtime
	logger            logging.ApplicationLogger
	ctx               context.Context
	cancel            context.CancelFunc
}

// Start runs the trading strategy
func (r *startup) Start(
	strategyPath string,
	connectors map[connector.ExchangeName]connector.Config,
	assets map[portfolio.Asset][]connector.Instrument,
) error {
	r.ctx, r.cancel = context.WithCancel(context.Background())

	bootConfig := runtime.BootConfig{
		StrategyPath:   strategyPath,
		ConnectorNames: make([]connector.ExchangeName, 0, len(connectors)),
	}

	for name, config := range connectors {
		conn, isRegistered := r.connectorRegistry.GetConnector(name)
		if !isRegistered {
			r.logger.Warn(fmt.Sprintf("connector %s is not registered", name))
		}

		err := conn.Initialize(config)
		if err != nil {
			r.logger.Error(fmt.Sprintf("connector %s initialize failed: %s", name, err.Error()))
			return err
		}

		bootConfig.ConnectorNames = append(bootConfig.ConnectorNames, name)
		err = r.connectorRegistry.MarkConnectorReady(name)
		if err != nil {
			return err
		}
	}

	for asset, instruments := range assets {
		for _, instr := range instruments {
			r.assetRegistry.RegisterAsset(asset, instr)
		}
	}

	err := r.runtime.Boot(r.ctx, bootConfig)
	if err != nil {
		r.logger.Error(fmt.Sprintf("runtime boot failed: %s", err.Error()))
		return err
	}

	return nil
}

// Stop gracefully shuts down the runtime
func (r *startup) Stop() error {
	r.logger.Info("stopping startup service")

	if r.cancel != nil {
		r.cancel()
	}

	return r.runtime.Stop(r.ctx)
}
