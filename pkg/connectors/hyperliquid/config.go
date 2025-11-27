package hyperliquid

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

type Config struct {
	BaseURL         string  `json:"base_url,omitempty"`
	PrivateKey      string  `json:"private_key"`
	AccountAddress  string  `json:"account_address"`
	VaultAddress    string  `json:"vault_address,omitempty"`
	UseTestnet      bool    `json:"use_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default slippage for market orders (0.005 = 0.5%)
}

var _ connector.Config = (*Config)(nil)

func (c *Config) ExchangeName() string {
	return "hyperliquid"
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	if c.AccountAddress == "" {
		return fmt.Errorf("account_address is required")
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
		return fmt.Errorf("default_slippage must be between 0 and 0.1 (0-10%%), got: %f", c.DefaultSlippage)
	}

	return nil
}
