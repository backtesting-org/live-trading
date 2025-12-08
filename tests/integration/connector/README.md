# Connector Integration Tests

Tests for exchange connector implementations.

## Quick Setup

1. **Copy the environment template:**
   ```bash
   cp .env.example .env
   ```

2. **Fill in your credentials in `.env`:**
   - For Hyperliquid: `HYPERLIQUID_ACCOUNT_ADDRESS` and `HYPERLIQUID_PRIVATE_KEY`
   - For Paradex: `PARADEX_ACCOUNT_ADDRESS` and `PARADEX_ETH_PRIVATE_KEY`
   - For Bybit: `BYBIT_API_KEY` and `BYBIT_API_SECRET`

3. **Choose which connector to test in `config_test.go`:**
   ```go
   const testConnectorName = types.Hyperliquid
   ```

4. **Run the tests:**
   ```bash
   go test -v ./tests/integration/connector/...
   ```

## Test Files

- `initialization_test.go` - Connector setup and capabilities
- `market_data_test.go` - REST API market data
- `account_data_test.go` - REST API account data
- `websocket_lifecycle_test.go` - WebSocket connection
- `websocket_subscription_test.go` - Real-time data streams
- `trading_operations_test.go` - Order placement (disabled by default)

## Running Specific Tests

```bash
# Run only market data tests
go test -v ./tests/integration/connector/ -ginkgo.focus="Market Data"

# Run only WebSocket tests
go test -v ./tests/integration/connector/ -ginkgo.focus="WebSocket"

# Skip trading tests
go test -v ./tests/integration/connector/ -ginkgo.skip="Trading"
```

## Configuration

Edit `config_test.go` to change:
- `testConnectorName` - Which connector to test (Hyperliquid, Paradex, Bybit)
- `testSymbol` - Asset symbol (default: "BTC")
- `testInstrumentType` - Instrument type (default: Perpetual)
- `enableTradingTests` - Enable order tests (default: false, **DANGEROUS**)

## Safety

- Trading tests are **disabled by default**
- Always use testnet credentials
- `.env` file is git-ignored for security
- Never commit real credentials

