package paradex

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

type Config struct {
	BaseURL        string
	WebSocketURL   string
	StarknetRPC    string
	AccountAddress string
	EthPrivateKey  string
	L2PrivateKey   string
	Network        string
}

var _ connector.Config = (*Config)(nil)

func (c *Config) Validate() error {
	if c.EthPrivateKey == "" {
		return fmt.Errorf("eth-private-key is required")
	}
	if c.AccountAddress == "" {
		return fmt.Errorf("account-address is required")
	}
	if c.Network == "" {
		c.Network = "mainnet"
	}

	// Set defaults based on network
	if c.BaseURL == "" {
		if c.Network == "testnet" {
			c.BaseURL = "https://api.testnet.paradex.trade/consumer"
		} else {
			c.BaseURL = "https://api.paradex.trade/consumer"
		}
	}

	if c.StarknetRPC == "" {
		if c.Network == "testnet" {
			c.StarknetRPC = "https://starknet-sepolia.public.blastapi.io"
		} else {
			c.StarknetRPC = "https://starknet-mainnet.public.blastapi.io"
		}
	}

	if c.WebSocketURL == "" {
		if c.Network == "testnet" {
			c.WebSocketURL = "wss://ws.testnet.paradex.trade/v1"
		} else {
			c.WebSocketURL = "wss://ws.paradex.trade/v1"
		}
	}

	return nil
}

func (c *Config) ExchangeName() string {
	return "paradex"
}
