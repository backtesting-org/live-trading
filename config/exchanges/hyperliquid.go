package exchange

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type HyperliquidConfig struct {
	BaseURL        string `mapstructure:"base_url" validate:"required,url"`
	PrivateKey     string `mapstructure:"private_key"`
	AccountAddress string `mapstructure:"account_address"`
	VaultAddress   string `mapstructure:"vault_address"`
	UseTestnet     bool   `mapstructure:"use_testnet"`
}

func (c *HyperliquidConfig) LoadHyperliquidConfig() {
	viper.Set("hyperliquid.base_url", os.Getenv("HYPERLIQUID_BASE_URL"))
	viper.Set("hyperliquid.private_key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"))
	viper.Set("hyperliquid.account_address", os.Getenv("HYPERLIQUID_ACCOUNT_ADDRESS"))
	viper.Set("hyperliquid.vault_address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"))
	viper.Set("hyperliquid.use_testnet", os.Getenv("HYPERLIQUID_USE_TESTNET"))

	if err := viper.UnmarshalKey("hyperliquid", c); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal hyperliquid config: %v", err))
	}

	err := c.Validate()
	if err != nil {
		panic(fmt.Sprintf("HyperliquidConfig validation failed: %v", err))
	}
}

func (c *HyperliquidConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
