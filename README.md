# Live Trading API

A live trading platform that supports automated trading strategies with real-time market data integration across multiple exchanges.

## Features

- **Exchange-Agnostic Architecture** - Supports multiple exchanges via adapter pattern
- **Plugin-Based Strategy System** - Load custom trading strategies dynamically
- **Real-Time Market Data** - Live price feeds, orderbooks, and klines
- **PostgreSQL Database** - Persistent storage for runs, signals, and logs
- **RESTful API** - Comprehensive strategy and account management
- **WebSocket Support** - Real-time updates and notifications

## Supported Exchanges

- ✅ **Paradex** - Perpetual futures on Starknet
- ✅ **Hyperliquid** - Decentralized perpetual DEX
- ✅ **Bybit** - Centralized crypto derivatives exchange

### Discover Available Exchanges

List all supported exchanges and their configuration requirements:

```bash
# Human-readable format
make build-cli
./bin/kronos-live list-exchanges

# JSON format (for programmatic use)
./bin/kronos-live list-exchanges --json
```

See [Exchange Metadata Documentation](docs/EXCHANGE_METADATA.md) for details on the metadata system.

## Prerequisites

- Go 1.24 or higher
- PostgreSQL database (Neon recommended)

## Setup

### 1. Environment Variables

Copy the example .env file and configure your settings:

```bash
cp .env.example .env
```

Configure the following required variables:

```env
# Database Configuration (Required)
LIVE_TRADING_DATABASE_CONNECTION_STRING="postgresql://user:pass@host/db?sslmode=require"
```

**Exchange-Specific Configuration:**

Each exchange requires its own configuration. See `.env.example` for details.

<details>
<summary><b>Paradex Configuration</b></summary>

```env
PARADEX_BASE_URL=https://api.testnet.paradex.trade/v1
PARADEX_WEBSOCKET_URL=wss://ws.api.testnet.paradex.trade/v1
PARADEX_STARKNET_RPC=https://rpc.api.testnet.paradex.trade/
PARADEX_ACCOUNT_ADDRESS=your_starknet_testnet_account_address_here
PARADEX_ETH_PRIVATE_KEY=your_ethereum_private_key_here
PARADEX_L2_PRIVATE_KEY=your_starknet_l2_private_key_here
PARADEX_USE_TESTNET=true
PARADEX_TIMEOUT=30s
```
</details>

<details>
<summary><b>Binance Configuration</b> (Coming Soon)</summary>

```env
BINANCE_API_KEY=your_api_key_here
BINANCE_API_SECRET=your_api_secret_here
BINANCE_USE_TESTNET=true
```
</details>

### 2. Exchange Account Setup

#### Paradex

Before starting the server with Paradex, ensure your Ethereum wallet (corresponding to `PARADEX_ETH_PRIVATE_KEY`) has:
- At least **0.001 ETH** OR **5 USDC**
- On **Ethereum, Arbitrum, or Base mainnet**

Without sufficient funds, onboarding will fail with:
```
INSUFFICIENT_MIN_CHAIN_BALANCE
```

#### Other Exchanges

See exchange-specific documentation in `.env.example`

### 3. Build and Run

```bash
# Build the server
make build

# Build plugins
make plugins

# Run the server
./bin/live-trading-api
```

## API Endpoints

### Health Check
```
GET /health
```

### Plugins
```
GET    /api/v1/plugins              # List all plugins
GET    /api/v1/plugins/:id          # Get plugin by ID
POST   /api/v1/plugins/upload       # Upload new plugin
DELETE /api/v1/plugins/:id          # Delete plugin
```

### Strategy Execution
```
POST   /api/v1/strategies/start           # Start a strategy
GET    /api/v1/strategies/runs            # List strategy runs (requires plugin_id param)
POST   /api/v1/strategies/:runId/stop     # Stop a running strategy
GET    /api/v1/strategies/:runId/status   # Get strategy status
GET    /api/v1/strategies/:runId/stats    # Get strategy statistics
```

### WebSocket
```
WS /ws
```

## CLI Usage

The `kronos-live` CLI allows you to run trading strategies directly from the command line without needing the API server.

### Build the CLI

```bash
make build-cli
```

### List Available Exchanges

```bash
# Human-readable format
./bin/kronos-live list-exchanges

# JSON format (for automation)
./bin/kronos-live list-exchanges --json
```

### Run a Strategy

```bash
# General syntax
./bin/kronos-live run --exchange <exchange> --strategy <path-to-plugin.so> [exchange-specific-flags]

# Example: Run with Hyperliquid
./bin/kronos-live run \
  --exchange hyperliquid \
  --strategy ./plugins/my-strategy.so \
  --hyperliquid-account-address 0x... \
  --hyperliquid-private-key 0x... \
  --hyperliquid-use-testnet false

# Example: Run with Bybit
./bin/kronos-live run \
  --exchange bybit \
  --strategy ./plugins/my-strategy.so \
  --bybit-api-key your-api-key \
  --bybit-api-secret your-api-secret \
  --bybit-is-testnet true
```

### Get Help for Specific Exchange

```bash
# See all available flags for an exchange
./bin/kronos-live run --exchange hyperliquid --help
```

See [Exchange Metadata Documentation](docs/EXCHANGE_METADATA.md) for more details on CLI integration.

## Development

### Project Structure

```
.
├── cmd/api/                     # Application entry point
├── config/
│   └── exchanges/               # Exchange-specific configurations
├── internal/                    # ✅ Exchange-agnostic core
│   ├── api/                     # HTTP handlers and routes
│   ├── connectors/              # Connector registry (generic)
│   ├── database/                # Database repository
│   └── services/                # Business logic (uses connector.Connector)
├── external/
│   └── exchanges/               # ✅ Exchange-specific implementations
│       └── paradex/             # Paradex connector & client
│           ├── connector.go     # Implements connector.Connector
│           ├── provider.go      # Factory function
│           ├── adaptor/         # HTTP client
│           ├── requests/        # API requests
│           └── websocket/       # WebSocket client
└── plugins/                     # Trading strategy plugins
```

### Architecture Principles

1. **`internal/`** - Exchange-agnostic, uses only `connector.Connector` interface
2. **`external/exchanges/`** - Exchange-specific implementations
3. **`cmd/api/main.go`** - Registers connectors with the registry
4. **Zero dependencies** - `internal/` never imports `external/exchanges/`

### Building Plugins

Strategy plugins must be built with the exact same version of `kronos-sdk` as the main server:

```bash
make plugins
```

## Adding a New Exchange

To add support for a new exchange (e.g., Binance):

1. **Create exchange adapter**:
   ```bash
   mkdir -p external/exchanges/binance
   ```

2. **Implement `connector.Connector` interface**:
   ```go
   // external/exchanges/binance/connector.go
   package binance

   type Binance struct { ... }

   func (b *Binance) FetchKlines(...) ([]connector.Kline, error) { ... }
   func (b *Binance) PlaceMarketOrder(...) (*connector.OrderResponse, error) { ... }
   // ... implement all connector.Connector methods
   ```

3. **Create provider function**:
   ```go
   // external/exchanges/binance/provider.go
   func NewConnector(config *Config, ...) (connector.Connector, error) {
       return &Binance{...}, nil
   }
   ```

4. **Register in `cmd/api/main.go`**:
   ```go
   registry.Register("binance", func() (connector.Connector, error) {
       return binance.NewConnector(binanceConfig, appLogger, tradingLogger)
   })
   ```

5. **Update `.env`**:
   ```env
   LIVE_TRADING_EXCHANGE_NAME=binance
   ```

**That's it!** No changes to `internal/` required.

## Troubleshooting

### Exchange Onboarding Fails (Paradex)

**Error:** `NOT_ONBOARDED` or `INSUFFICIENT_MIN_CHAIN_BALANCE`

**Solution:**
1. Verify your Ethereum wallet has at least 0.001 ETH or 5 USDC on Ethereum, Arbitrum, or Base mainnet
2. Testnet funds (Sepolia, Goerli, etc.) will NOT work
3. Restart the server after funding your wallet

### PostgreSQL Prepared Statement Errors

**Error:** `bind message supplies X parameters, but prepared statement requires Y`

**Solution:**
- This has been fixed by using raw SQL queries instead of prepared statements
- If you still encounter this, restart the server to clear any cached connections

### Plugin Version Mismatch

**Error:** `plugin was built with a different version of package kronos-sdk`

**Solution:**
```bash
# Update kronos-sdk and rebuild both server and plugins
go get github.com/backtesting-org/kronos-sdk@latest
make build
make plugins
```

## License

[Your License Here]
