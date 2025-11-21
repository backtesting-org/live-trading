package handlers

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
)

// GetConnectorFromRegistry retrieves a connector from the SDK registry
func GetConnectorFromRegistry(reg registry.ConnectorRegistry, exchangeName connector.ExchangeName) (connector.Connector, error) {
	conn, found := reg.GetConnector(exchangeName)
	if !found {
		return nil, fmt.Errorf("connector for exchange %s not found in registry", exchangeName)
	}

	return conn, nil
}
