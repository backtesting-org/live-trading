package hyperliquid

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

type Config struct {
	BaseURL         string
	PrivateKey      string
	AccountAddress  string
	VaultAddress    string
	UseTestnet      bool
	DefaultSlippage float64 // Default slippage for market orders (0.005 = 0.5%)
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

	// Set default slippage if not specified (0.5%)
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	// Validate slippage is reasonable (0-10%)
	if c.DefaultSlippage < 0 || c.DefaultSlippage > 0.1 {
		return fmt.Errorf("default slippage must be between 0 and 0.1 (0-10%%), got: %f", c.DefaultSlippage)
	}

	return nil
}
