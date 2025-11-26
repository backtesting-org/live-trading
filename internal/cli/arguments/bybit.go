package arguments

import (
	"github.com/backtesting-org/live-trading/external/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type BybitArguments struct{}

func NewBybitArguments() *BybitArguments {
	return &BybitArguments{}
}

func (p *BybitArguments) ExchangeName() string {
	return "bybit"
}

func (p *BybitArguments) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("bybit-api-key", "", "Bybit API key")
	cmd.Flags().String("bybit-api-secret", "", "Bybit API secret")
	cmd.Flags().Bool("bybit-is-testnet", false, "Use Bybit testnet")
	cmd.Flags().Float64("bybit-default-slippage", 0.005, "Default slippage percentage (e.g., 0.005 for 0.5%)")
}

func (p *BybitArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	APIKey, _ := cmd.Flags().GetString("bybit-api-key")
	APISecret, _ := cmd.Flags().GetString("bybit-api-secret")
	IsTestnet, _ := cmd.Flags().GetBool("bybit-is-testnet")
	DefaultSlippage, _ := cmd.Flags().GetFloat64("bybit-default-slippage")

	cfg := &bybit.Config{
		APIKey:          APIKey,
		APISecret:       APISecret,
		IsTestnet:       IsTestnet,
		DefaultSlippage: DefaultSlippage,
	}

	return cfg, cfg.Validate()
}
