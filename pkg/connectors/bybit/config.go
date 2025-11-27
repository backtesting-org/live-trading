package bybit

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

// Config holds the configuration for the Bybit connector
type Config struct {
	APIKey          string  `json:"api_key"`
	APISecret       string  `json:"api_secret"`
	BaseURL         string  `json:"base_url,omitempty"`
	IsTestnet       bool    `json:"is_testnet,omitempty"`
	DefaultSlippage float64 `json:"default_slippage,omitempty"` // Default 0.005 (0.5%)
}

var _ connector.Config = (*Config)(nil)

func (c *Config) ExchangeName() string {
	return "bybit"
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.APISecret == "" {
		return fmt.Errorf("api_secret is required")
	}

	// Set default slippage if not provided
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	// Set base URL based on testnet flag if not explicitly provided
	if c.BaseURL == "" {
		if c.IsTestnet {
			c.BaseURL = "https://api-testnet.bybit.com"
		} else {
			c.BaseURL = "https://api.bybit.com"
		}
	}

	return nil
}
