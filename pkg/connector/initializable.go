package connector

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
)

// Initializable extends Connector with runtime initialization capability
type Initializable interface {
	connector.Connector

	// Initialize configures the connector with exchange-specific settings at runtime
	Initialize(config Config) error

	// IsInitialized returns whether the connector has been initialized
	IsInitialized() bool
}
