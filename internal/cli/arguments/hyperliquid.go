package arguments

import (
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/spf13/cobra"
)

type HyperliquidArguments struct {
	metadata *ExchangeMetadata
}

func NewHyperliquidArguments() *HyperliquidArguments {
	return &HyperliquidArguments{
		metadata: &ExchangeMetadata{
			Name: "hyperliquid",
			Flags: []FlagMetadata{
				{
					Name:         "hyperliquid-private-key",
					Type:         FlagTypeString,
					Description:  "Hyperliquid private key",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "hyperliquid-account-address",
					Type:         FlagTypeString,
					Description:  "Hyperliquid account address",
					DefaultValue: "",
					Required:     true,
				},
				{
					Name:         "hyperliquid-vault-address",
					Type:         FlagTypeString,
					Description:  "Hyperliquid vault address (optional)",
					DefaultValue: "",
					Required:     false,
				},
				{
					Name:         "hyperliquid-use-testnet",
					Type:         FlagTypeBool,
					Description:  "Use Hyperliquid testnet",
					DefaultValue: false,
					Required:     false,
				},
			},
		},
	}
}

func (h *HyperliquidArguments) ExchangeName() string {
	return h.metadata.Name
}

func (h *HyperliquidArguments) Metadata() *ExchangeMetadata {
	return h.metadata
}

func (h *HyperliquidArguments) RegisterFlags(cmd *cobra.Command) {
	_ = h.metadata.RegisterFlags(cmd)
}

func (h *HyperliquidArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	values, err := h.metadata.GetFlagValues(cmd)
	if err != nil {
		return nil, err
	}

	cfg := &hyperliquid.Config{
		PrivateKey:     values["hyperliquid-private-key"].(string),
		AccountAddress: values["hyperliquid-account-address"].(string),
		VaultAddress:   values["hyperliquid-vault-address"].(string),
		UseTestnet:     values["hyperliquid-use-testnet"].(bool),
	}

	return cfg, cfg.Validate()
}
