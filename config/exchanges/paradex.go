package exchange

import (
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Paradex struct {
	BaseURL        string        `mapstructure:"base_url" validate:"required,url"`
	WebSocketURL   string        `mapstructure:"websocket_url"`
	StarknetRPC    string        `mapstructure:"starknet_rpc" validate:"required,url"`
	AccountAddress string        `mapstructure:"account_address"`
	EthPrivateKey  string        `mapstructure:"eth_private_key" validate:"required"`
	L2PrivateKey   string        `mapstructure:"l2_private_key"`
	Timeout        time.Duration `mapstructure:"timeout"`
	UseTestnet     bool          `mapstructure:"use_testnet"`
}

func (c *Paradex) LoadParadexConfig() {
	viper.Set("paradex.base_url", os.Getenv("PARADEX_BASE_URL"))
	viper.Set("paradex.websocket_url", os.Getenv("PARADEX_WEBSOCKET_URL"))
	viper.Set("paradex.starknet_rpc", os.Getenv("PARADEX_STARKNET_RPC"))
	viper.Set("paradex.account_address", os.Getenv("PARADEX_ACCOUNT_ADDRESS"))
	viper.Set("paradex.eth_private_key", os.Getenv("PARADEX_ETH_PRIVATE_KEY"))
	viper.Set("paradex.l2_private_key", os.Getenv("PARADEX_L2_PRIVATE_KEY"))
	viper.Set("paradex.timeout", os.Getenv("PARADEX_TIMEOUT"))
	viper.Set("paradex.use_testnet", os.Getenv("PARADEX_USE_TESTNET"))

	// Set defaults
	if viper.GetString("paradex.timeout") == "" {
		viper.Set("paradex.timeout", "30s")
	}
	if viper.GetString("paradex.base_url") == "" {
		if viper.GetBool("paradex.use_testnet") {
			viper.Set("paradex.base_url", "https://api.testnet.paradex.io/consumer")
		} else {
			viper.Set("paradex.base_url", "https://api.paradex.trade/consumer")
		}
	}
	if viper.GetString("paradex.starknet_rpc") == "" {
		if viper.GetBool("paradex.use_testnet") {
			viper.Set("paradex.starknet_rpc", "https://starknet-sepolia.public.blastapi.io")
		} else {
			viper.Set("paradex.starknet_rpc", "https://starknet-mainnet.public.blastapi.io")
		}
	}

	if err := viper.UnmarshalKey("paradex", c); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal paradex config: %v", err))
	}

	err := c.Validate()
	if err != nil {
		panic(fmt.Sprintf("ParadexConfig validation failed: %v", err))
	}
}

func (c *Paradex) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
