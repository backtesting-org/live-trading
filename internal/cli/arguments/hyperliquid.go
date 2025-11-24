package arguments

import (
	"github.com/backtesting-org/live-trading/external/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type HyperliquidArguments struct{}

func NewHyperliquidArguments() *HyperliquidArguments {
	return &HyperliquidArguments{}
}

func (h *HyperliquidArguments) ExchangeName() string {
	return "hyperliquid"
}

func (h *HyperliquidArguments) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("hyperliquid-private-key", "", "Hyperliquid private key")
	cmd.Flags().String("hyperliquid-account-address", "", "Hyperliquid account address")
	cmd.Flags().String("hyperliquid-vault-address", "", "Hyperliquid vault address (optional)")
	cmd.Flags().Bool("hyperliquid-use-testnet", false, "Use Hyperliquid testnet")
}

func (h *HyperliquidArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	privateKey, _ := cmd.Flags().GetString("hyperliquid-private-key")
	accountAddress, _ := cmd.Flags().GetString("hyperliquid-account-address")
	vaultAddress, _ := cmd.Flags().GetString("hyperliquid-vault-address")
	useTestnet, _ := cmd.Flags().GetBool("hyperliquid-use-testnet")

	cfg := &hyperliquid.Config{
		PrivateKey:     privateKey,
		AccountAddress: accountAddress,
		VaultAddress:   vaultAddress,
		UseTestnet:     useTestnet,
	}

	return cfg, cfg.Validate()
}
