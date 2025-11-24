package clients

import (
	"fmt"

	exchange "github.com/backtesting-org/live-trading/config/exchanges"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

func NewExchangeClient(cfg *exchange.HyperliquidConfig) (*hyperliquid.Exchange, error) {
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("private key required for exchange client")
	}

	privateKey, err := crypto.HexToECDSA(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return hyperliquid.NewExchange(
		privateKey,
		cfg.BaseURL,
		nil,
		cfg.VaultAddress,
		cfg.AccountAddress,
		nil,
	), nil
}
