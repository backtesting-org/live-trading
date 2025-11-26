package arguments

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type ArgumentHandler interface {
	RegisterFlags(cmd *cobra.Command)
	ParseConfig(cmd *cobra.Command) (connector.Config, error)
	ExchangeName() string
	Metadata() *ExchangeMetadata
}

type Registry struct {
	handlers map[string]ArgumentHandler
}

func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]ArgumentHandler),
	}
	r.Register(NewParadexArguments())
	r.Register(NewHyperliquidArguments())
	r.Register(NewBybitArguments())
	return r
}

func (r *Registry) Register(handler ArgumentHandler) {
	r.handlers[handler.ExchangeName()] = handler
}

func (r *Registry) GetHandler(exchange string) (ArgumentHandler, error) {
	handler, ok := r.handlers[exchange]
	if !ok {
		return nil, fmt.Errorf("unsupported exchange: %s", exchange)
	}
	return handler, nil
}

func (r *Registry) RegisterAllFlags(cmd *cobra.Command) {
	for _, handler := range r.handlers {
		handler.RegisterFlags(cmd)
	}
}

// GetAllMetadata returns metadata for all registered exchanges
func (r *Registry) GetAllMetadata() []*ExchangeMetadata {
	metadata := make([]*ExchangeMetadata, 0, len(r.handlers))
	for _, handler := range r.handlers {
		metadata = append(metadata, handler.Metadata())
	}
	return metadata
}

// GetExchangeNames returns a list of all registered exchange names
func (r *Registry) GetExchangeNames() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}
