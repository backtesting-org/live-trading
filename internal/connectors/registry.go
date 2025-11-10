package connectors

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
)

// ProviderFunc is a function that creates a connector instance
type ProviderFunc func() (connector.Connector, error)

// Registry holds registered connector providers
type Registry struct {
	providers map[string]ProviderFunc
}

// NewRegistry creates a new connector registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ProviderFunc),
	}
}

// Register adds a connector provider for a given exchange name
func (r *Registry) Register(exchangeName string, provider ProviderFunc) {
	r.providers[exchangeName] = provider
}

// GetConnector creates a connector for the specified exchange
func (r *Registry) GetConnector(exchangeName string) (connector.Connector, error) {
	provider, exists := r.providers[exchangeName]
	if !exists {
		return nil, fmt.Errorf("no connector provider registered for exchange: %s", exchangeName)
	}

	return provider()
}

// ListExchanges returns all registered exchange names
func (r *Registry) ListExchanges() []string {
	exchanges := make([]string, 0, len(r.providers))
	for name := range r.providers {
		exchanges = append(exchanges, name)
	}
	return exchanges
}
