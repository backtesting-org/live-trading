package arguments

import (
	"github.com/backtesting-org/live-trading/external/connectors/paradex"
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type ParadexArguments struct {
	metadata *ExchangeMetadata
}

func NewParadexArguments() *ParadexArguments {
	return &ParadexArguments{
		metadata: &ExchangeMetadata{
			Name: "paradex",
			Flags: []FlagMetadata{
				{
					Name:         "paradex-eth-private-key",
					Type:         FlagTypeString,
					Description:  "Paradex Ethereum private key",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "paradex-l2-private-key",
					Type:         FlagTypeString,
					Description:  "Paradex L2 private key (optional)",
					DefaultValue: "",
					Required:     false,
				},
				{
					Name:         "paradex-account-address",
					Type:         FlagTypeString,
					Description:  "Paradex account address",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "paradex-network",
					Type:         FlagTypeString,
					Description:  "Paradex network (mainnet/testnet)",
					DefaultValue: "mainnet",
					Required:     false,
				},
				{
					Name:         "paradex-base-url",
					Type:         FlagTypeString,
					Description:  "Paradex API base URL (optional, auto-detected)",
					DefaultValue: "",
					Required:     false,
				},
				{
					Name:         "paradex-ws-url",
					Type:         FlagTypeString,
					Description:  "Paradex WebSocket URL (optional, auto-detected)",
					DefaultValue: "",
					Required:     false,
				},
				{
					Name:         "paradex-starknet-rpc",
					Type:         FlagTypeString,
					Description:  "Starknet RPC URL (optional, auto-detected)",
					DefaultValue: "",
					Required:     false,
				},
			},
		},
	}
}

func (p *ParadexArguments) ExchangeName() string {
	return p.metadata.Name
}

func (p *ParadexArguments) Metadata() *ExchangeMetadata {
	return p.metadata
}

func (p *ParadexArguments) RegisterFlags(cmd *cobra.Command) {
	_ = p.metadata.RegisterFlags(cmd)
}

func (p *ParadexArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	values, err := p.metadata.GetFlagValues(cmd)
	if err != nil {
		return nil, err
	}

	cfg := &paradex.Config{
		EthPrivateKey:  values["paradex-eth-private-key"].(string),
		L2PrivateKey:   values["paradex-l2-private-key"].(string),
		AccountAddress: values["paradex-account-address"].(string),
		Network:        values["paradex-network"].(string),
		BaseURL:        values["paradex-base-url"].(string),
		WebSocketURL:   values["paradex-ws-url"].(string),
		StarknetRPC:    values["paradex-starknet-rpc"].(string),
	}

	return cfg, cfg.Validate()
}
