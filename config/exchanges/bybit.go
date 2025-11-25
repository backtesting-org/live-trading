package exchange

import (
	"os"
)

type BybitConfig struct {
	APIKey    string
	APISecret string
	Testnet   bool
}

func (c *BybitConfig) LoadBybitConfig() {
	c.APIKey = os.Getenv("BYBIT_API_KEY")
	c.APISecret = os.Getenv("BYBIT_API_SECRET")
	c.Testnet = os.Getenv("BYBIT_TESTNET") == "true"
}
