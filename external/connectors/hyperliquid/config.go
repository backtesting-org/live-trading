package hyperliquid

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

type Config struct {
	BaseURL        string
	PrivateKey     string
	AccountAddress string
	VaultAddress   string
	UseTestnet     bool
}

var _ connector.Config = (*Config)(nil)

func (c *Config) ExchangeName() string {
	return "hyperliquid"
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return fmt.Errorf("private-key is required")
	}
	if c.AccountAddress == "" {
		return fmt.Errorf("account-address is required")
	}

	// Set default base URL based on network
	if c.BaseURL == "" {
		if c.UseTestnet {
			c.BaseURL = "https://api.hyperliquid-testnet.xyz"
		} else {
			c.BaseURL = "https://api.hyperliquid.xyz"
		}
	}

	return nil
}
