package connector_test

import (
	"os"
	"path/filepath"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/live-trading/pkg/connectors/bybit"
	"github.com/backtesting-org/live-trading/pkg/connectors/hyperliquid"
	"github.com/backtesting-org/live-trading/pkg/connectors/paradex"
	"github.com/backtesting-org/live-trading/pkg/connectors/types"
	"github.com/joho/godotenv"
)

func init() {
	// Try to load .env file from the test directory
	// Ignore errors if file doesn't exist (env vars may be set directly)
	envPath := filepath.Join(".", ".env")
	_ = godotenv.Load(envPath)
}

// ========================================
// TEST CONFIGURATION - EDIT HERE
// ========================================
const (
	// Which connector to test
	testConnectorName = types.Hyperliquid // Change to types.Paradex or types.Bybit

	// Test asset
	testSymbol = "BTC"

	// Test instrument type
	testInstrumentType = connector.TypePerpetual

	// Enable trading tests (DANGEROUS - only on testnet)
	enableTradingTests = false
)

// ========================================
// CONFIG LOADERS
// ========================================

func getConnectorConfig(name connector.ExchangeName) connector.Config {
	switch name {
	case types.Hyperliquid:
		return getHyperliquidConfig()
	case types.Paradex:
		return getParadexConfig()
	case types.Bybit:
		return getBybitConfig()
	default:
		panic("unknown connector: " + name)
	}
}

func getHyperliquidConfig() connector.Config {
	return &hyperliquid.Config{
		BaseURL:        getEnv("HYPERLIQUID_BASE_URL", "https://api.hyperliquid.xyz"),
		AccountAddress: mustGetEnv("HYPERLIQUID_ACCOUNT_ADDRESS"),
		PrivateKey:     mustGetEnv("HYPERLIQUID_PRIVATE_KEY"),
		VaultAddress:   getEnv("HYPERLIQUID_VAULT_ADDRESS", ""),
		UseTestnet:     getEnv("HYPERLIQUID_TESTNET", "false") == "true",
	}
}

func getParadexConfig() connector.Config {
	return &paradex.Config{
		BaseURL:        getEnv("PARADEX_BASE_URL", "https://api.testnet.paradex.trade/consumer"),
		WebSocketURL:   getEnv("PARADEX_WS_URL", "wss://ws.testnet.paradex.trade/v1"),
		StarknetRPC:    getEnv("PARADEX_STARKNET_RPC", "https://starknet-sepolia.public.blastapi.io"),
		AccountAddress: mustGetEnv("PARADEX_ACCOUNT_ADDRESS"),
		EthPrivateKey:  mustGetEnv("PARADEX_ETH_PRIVATE_KEY"),
		Network:        getEnv("PARADEX_NETWORK", "testnet"),
	}
}

func getBybitConfig() connector.Config {
	return &bybit.Config{
		APIKey:    mustGetEnv("BYBIT_API_KEY"),
		APISecret: mustGetEnv("BYBIT_API_SECRET"),
		IsTestnet: getEnv("BYBIT_TESTNET", "true") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic("required environment variable not set: " + key)
	}
	return value
}
