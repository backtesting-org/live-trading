package arguments

import (
	"github.com/backtesting-org/live-trading/external/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type BybitArguments struct {
	metadata *ExchangeMetadata
}

func NewBybitArguments() *BybitArguments {
	return &BybitArguments{
		metadata: &ExchangeMetadata{
			Name: "bybit",
			Flags: []FlagMetadata{
				{
					Name:         "bybit-api-key",
					Type:         FlagTypeString,
					Description:  "Bybit API key",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "bybit-api-secret",
					Type:         FlagTypeString,
					Description:  "Bybit API secret",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "bybit-is-testnet",
					Type:         FlagTypeBool,
					Description:  "Use Bybit testnet",
					DefaultValue: false,
					Required:     false,
				},
				{
					Name:         "bybit-default-slippage",
					Type:         FlagTypeFloat64,
					Description:  "Default slippage percentage (e.g., 0.005 for 0.5%)",
					DefaultValue: 0.005,
					Required:     false,
				},
			},
		},
	}
}

func (p *BybitArguments) ExchangeName() string {
	return p.metadata.Name
}

func (p *BybitArguments) Metadata() *ExchangeMetadata {
	return p.metadata
}

func (p *BybitArguments) RegisterFlags(cmd *cobra.Command) {
	_ = p.metadata.RegisterFlags(cmd)
}

func (p *BybitArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	values, err := p.metadata.GetFlagValues(cmd)
	if err != nil {
		return nil, err
	}

	cfg := &bybit.Config{
		APIKey:          values["bybit-api-key"].(string),
		APISecret:       values["bybit-api-secret"].(string),
		IsTestnet:       values["bybit-is-testnet"].(bool),
		DefaultSlippage: values["bybit-default-slippage"].(float64),
	}

	return cfg, cfg.Validate()
}
