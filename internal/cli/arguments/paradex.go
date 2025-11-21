package arguments

import (
	"github.com/backtesting-org/live-trading/external/connectors/paradex"
	"github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
)

type ParadexArguments struct{}

func NewParadexArguments() *ParadexArguments {
	return &ParadexArguments{}
}

func (p *ParadexArguments) ExchangeName() string {
	return "paradex"
}

func (p *ParadexArguments) RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().String("paradex-eth-private-key", "", "Paradex Ethereum private key")
	cmd.Flags().String("paradex-l2-private-key", "", "Paradex L2 private key (optional)")
	cmd.Flags().String("paradex-account-address", "", "Paradex account address")
	cmd.Flags().String("paradex-network", "mainnet", "Paradex network (mainnet/testnet)")
	cmd.Flags().String("paradex-base-url", "", "Paradex API base URL (optional, auto-detected)")
	cmd.Flags().String("paradex-ws-url", "", "Paradex WebSocket URL (optional, auto-detected)")
	cmd.Flags().String("paradex-starknet-rpc", "", "Starknet RPC URL (optional, auto-detected)")
}

func (p *ParadexArguments) ParseConfig(cmd *cobra.Command) (connector.Config, error) {
	ethPrivateKey, _ := cmd.Flags().GetString("paradex-eth-private-key")
	l2PrivateKey, _ := cmd.Flags().GetString("paradex-l2-private-key")
	accountAddress, _ := cmd.Flags().GetString("paradex-account-address")
	network, _ := cmd.Flags().GetString("paradex-network")
	baseURL, _ := cmd.Flags().GetString("paradex-base-url")
	wsURL, _ := cmd.Flags().GetString("paradex-ws-url")
	starknetRPC, _ := cmd.Flags().GetString("paradex-starknet-rpc")

	cfg := &paradex.Config{
		EthPrivateKey:  ethPrivateKey,
		L2PrivateKey:   l2PrivateKey,
		AccountAddress: accountAddress,
		Network:        network,
		BaseURL:        baseURL,
		WebSocketURL:   wsURL,
		StarknetRPC:    starknetRPC,
	}

	return cfg, cfg.Validate()
}
