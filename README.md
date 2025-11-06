# Live Trading API

A live trading platform that supports automated trading strategies with real-time market data integration.

## Features

- Plugin-based trading strategy system
- Real-time market data feed via Paradex WebSocket
- PostgreSQL database for strategy runs, signals, and logs
- RESTful API for strategy management
- WebSocket support for real-time updates

## Prerequisites

- Go 1.24 or higher
- PostgreSQL database (Neon recommended)
- **Paradex Account Requirements:**
  - **IMPORTANT:** Your Ethereum wallet must contain at least **0.001 ETH or 5 USDC** on one of these mainnet chains:
    - Ethereum mainnet
    - Arbitrum
    - Base
  - This is a Paradex anti-spam requirement for account onboarding
  - Testnet funds (e.g., Sepolia ETH) will **NOT** work

## Setup

### 1. Environment Variables

Copy the example .env file and configure your settings:

```bash
cp .env.example .env
```

Configure the following required variables:

```env
# Database Configuration
LIVE_TRADING_DATABASE_CONNECTION_STRING="postgresql://user:pass@host/db?sslmode=require"

# Paradex Configuration
PARADEX_BASE_URL=https://api.testnet.paradex.trade/v1
PARADEX_WEBSOCKET_URL=wss://ws.api.testnet.paradex.trade/v1
PARADEX_STARKNET_RPC=https://rpc.api.testnet.paradex.trade/
PARADEX_ACCOUNT_ADDRESS=0x... # Your Paradex L2 address
PARADEX_ETH_PRIVATE_KEY=... # Your Ethereum private key
PARADEX_L2_PRIVATE_KEY=0x... # Your StarkNet private key
PARADEX_USE_TESTNET=true
```

### 2. Fund Your Ethereum Wallet

Before starting the server, ensure your Ethereum wallet (the one corresponding to `PARADEX_ETH_PRIVATE_KEY`) has:
- At least **0.001 ETH** OR **5 USDC**
- On **Ethereum, Arbitrum, or Base mainnet**

Without sufficient funds, the server will fail to onboard to Paradex with the error:
```
INSUFFICIENT_MIN_CHAIN_BALANCE
```

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

## Development

### Project Structure

```
.
├── cmd/api/                  # Application entry point
├── config/                   # Configuration files
├── internal/
│   ├── api/                  # HTTP handlers and routes
│   ├── database/             # Database repository
│   ├── services/             # Business logic
│   └── exchanges/            # Exchange integrations
├── external/
│   └── exchanges/paradex/    # Paradex client SDK
└── plugins/                  # Trading strategy plugins
```

### Building Plugins

Strategy plugins must be built with the exact same version of `kronos-sdk` as the main server:

```bash
make plugins
```

## Troubleshooting

### Paradex Onboarding Fails

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
