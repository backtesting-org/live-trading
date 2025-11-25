package bybit

import (
	"fmt"

	"github.com/backtesting-org/live-trading/pkg/connector"
)

// Config holds the configuration for the Bybit connector
type Config struct {
	APIKey          string
	APISecret       string
	BaseURL         string
	IsTestnet       bool
	DefaultSlippage float64 // Default 0.005 (0.5%)
}

var _ connector.Config = (*Config)(nil)

func (c *Config) ExchangeName() string {
	return "bybit"
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	if c.APISecret == "" {
		return fmt.Errorf("API secret is required")
	}

	// Set default slippage if not provided
	if c.DefaultSlippage == 0 {
		c.DefaultSlippage = 0.005
	}

	// Validate slippage range
	if c.DefaultSlippage < 0 || c.DefaultSlippage > 0.1 {
		return fmt.Errorf("default slippage must be between 0 and 0.1")
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
