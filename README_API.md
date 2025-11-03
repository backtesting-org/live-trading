# Live Trading API

A REST API server for managing and executing trading strategy plugins with real-time WebSocket updates.

## Features

- **Plugin Management**: Upload, list, and manage `.so` trading strategy plugins
- **Strategy Execution**: Start, stop, and monitor strategy execution
- **Real-time Updates**: WebSocket streaming of signals, trades, and events
- **Database Persistence**: Store plugins, configurations, runs, and signals in PostgreSQL (Neon)
- **Scalable Architecture**: Event-driven design with concurrent strategy execution

## Architecture

```
┌─────────────────┐
│  Frontend (UI)  │
└────────┬────────┘
         │ HTTP/WebSocket
         ↓
┌─────────────────────────────────────┐
│      Live Trading API Server        │
│  ┌──────────────────────────────┐  │
│  │  REST API (Gin)              │  │
│  │  - Plugin Upload/Management  │  │
│  │  - Strategy Execution        │  │
│  │  - Run Status & Stats        │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │  WebSocket Server            │  │
│  │  - Real-time event streaming │  │
│  └──────────────────────────────┘  │
│  ┌──────────────────────────────┐  │
│  │  Services Layer              │  │
│  │  - Plugin Manager            │  │
│  │  - Strategy Executor         │  │
│  │  - Event Bus                 │  │
│  └──────────────────────────────┘  │
└─────────────┬───────────────────────┘
              │
              ↓
      ┌───────────────┐
      │  Neon DB      │
      │  (PostgreSQL) │
      └───────────────┘
```

## Prerequisites

- Go 1.24.2 or higher
- PostgreSQL database (Neon recommended)
- Make (optional, for building plugins)

## Quick Start

### 1. Set up Database

Create a Neon PostgreSQL database and get your connection string:
```
postgres://user:password@host/database?sslmode=require
```

### 2. Configure Environment

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` and set your database connection string:
```bash
LIVE_TRADING_DATABASE_CONNECTION_STRING=your_neon_connection_string
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run Migrations

Migrations are run automatically on server startup. The schema includes:
- `plugin_metadata` - Uploaded plugins and their metadata
- `strategy_configs` - Strategy configurations (versioned)
- `strategy_runs` - Execution history
- `trading_signals` - Generated trading signals
- `execution_logs` - Execution logs and errors

### 5. Start the Server

```bash
go run cmd/api/main.go
```

The server will start on `http://localhost:8081` by default.

## API Endpoints

### Health Check

```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "service": "live-trading-api",
  "version": "1.0.0"
}
```

### Plugin Management

#### Upload Plugin
```http
POST /api/v1/plugins/upload
Content-Type: multipart/form-data

Fields:
- plugin: .so file (required)
- created_by: string (optional, default: "system")
```

Response:
```json
{
  "message": "Plugin uploaded successfully",
  "data": {
    "id": "uuid",
    "name": "Grid Trading",
    "description": "Market-neutral grid trading strategy",
    "risk_level": "medium",
    "type": "technical",
    "parameters": { ... }
  }
}
```

#### List Plugins
```http
GET /api/v1/plugins?limit=50&offset=0
```

#### Get Plugin
```http
GET /api/v1/plugins/:id
```

#### Delete Plugin
```http
DELETE /api/v1/plugins/:id
```

### Strategy Execution

#### Start Strategy
```http
POST /api/v1/strategies/start
Content-Type: application/json

{
  "plugin_id": "uuid",
  "config_id": "uuid" (optional)
}
```

Response:
```json
{
  "message": "Strategy started successfully",
  "data": {
    "run_id": "uuid",
    "plugin_id": "uuid"
  }
}
```

#### Stop Strategy
```http
POST /api/v1/strategies/:runId/stop
```

#### Get Run Status
```http
GET /api/v1/strategies/:runId/status
```

Response:
```json
{
  "message": "Run status retrieved successfully",
  "data": {
    "run_id": "uuid",
    "plugin_id": "uuid",
    "status": "running",
    "start_time": "2025-01-01T00:00:00Z",
    "total_signals": 150,
    "total_trades": 75,
    "error_count": 0,
    "cpu_usage": 0.0,
    "memory_usage": 52428800
  }
}
```

#### List Runs
```http
GET /api/v1/strategies/runs?plugin_id=uuid&limit=50&offset=0
```

Response:
```json
{
  "message": "Runs retrieved successfully",
  "data": {
    "runs": [
      {
        "id": "uuid",
        "plugin_id": "uuid",
        "status": "stopped",
        "start_time": "2025-01-01T00:00:00Z",
        "end_time": "2025-01-01T01:00:00Z",
        "total_signals": 200,
        "total_trades": 100,
        "profit_loss": "150.50",
        "error_count": 0
      }
    ],
    "count": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### Get Run Statistics
```http
GET /api/v1/strategies/:runId/stats
```

Response:
```json
{
  "message": "Run stats retrieved successfully",
  "data": {
    "total_signals": 150,
    "executed_signals": 75,
    "buy_signals": 80,
    "sell_signals": 70,
    "error_count": 0
  }
}
```

### WebSocket

Connect to real-time event stream:
```javascript
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data);
};
```

Event types:
- `strategy_started` - Strategy execution started
- `strategy_stopped` - Strategy execution stopped
- `strategy_error` - Strategy error occurred
- `signal_generated` - New trading signal generated
- `order_placed` - Order placed on exchange
- `order_filled` - Order filled
- `trade_executed` - Trade executed
- `account_update` - Account balance/position update

Example event:
```json
{
  "type": "signal_generated",
  "data": {
    "run_id": "uuid",
    "signal_id": "uuid",
    "signal_type": "BUY",
    "asset": "BTC",
    "exchange": "Paradex",
    "quantity": "0.1",
    "price": "50000.00",
    "timestamp": "2025-01-01T00:00:00Z"
  }
}
```

## Building and Uploading Plugins

### 1. Build a Plugin

Navigate to the plugin directory:
```bash
cd plugin-examples/grid
```

Build the plugin:
```bash
go build -buildmode=plugin -o grid.so .
```

**Important**: The plugin must be built with the exact same Go version (1.24.2) as the server.

### 2. Upload via API

```bash
curl -X POST http://localhost:8081/api/v1/plugins/upload \
  -F "plugin=@grid.so" \
  -F "created_by=admin"
```

### 3. Start the Strategy

Get the plugin ID from the upload response, then:
```bash
curl -X POST http://localhost:8081/api/v1/strategies/start \
  -H "Content-Type: application/json" \
  -d '{
    "plugin_id": "your-plugin-id"
  }'
```

## Plugin Development

Plugins must implement the `strategy.Strategy` interface from `kronos-sdk`:

```go
package main

import (
    "github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
)

type MyStrategy struct {
    *strategy.BaseStrategy
    // Your fields
}

func NewStrategy() strategy.Strategy {
    base := strategy.NewBaseStrategy(
        strategy.StrategyName("My Strategy"),
        "Description of my strategy",
        strategy.RiskLevelMedium,
        strategy.StrategyTypeTechnical,
    )

    return &MyStrategy{
        BaseStrategy: base,
    }
}

func (s *MyStrategy) GetSignals() ([]*strategy.Signal, error) {
    // Generate trading signals
    return signals, nil
}

// Export for plugin system
var Strategy strategy.Strategy
```

## Configuration

Configuration is set via **environment variables only** (prefix: `LIVE_TRADING_`).

See `.env.example` for all available options.

## Database Schema

The database schema is automatically created on first run. Key tables:

- **plugin_metadata**: Stores plugin metadata and parameters
- **strategy_configs**: Versioned strategy configurations
- **strategy_runs**: Execution history and metrics
- **trading_signals**: Generated trading signals
- **execution_logs**: Execution logs and errors

## Development

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -o live-trading-api cmd/api/main.go
```

### Docker Deployment (Optional)
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/internal/database/migrations ./internal/database/migrations
EXPOSE 8081
CMD ["./server"]
```

## Security Considerations

- **Authentication**: Not yet implemented - add JWT or API key authentication before production
- **CORS**: Currently set to allow all origins - restrict in production
- **File Upload**: Validates file extension and attempts to load plugin to verify validity
- **SQL Injection**: Using parameterized queries throughout
- **Plugin Isolation**: Go plugins run in the same process - implement sandboxing for production

## Monitoring

Metrics available via logs:
- Request latency
- Error rates
- Active WebSocket connections
- Strategy execution metrics
- Database connection pool stats

## Troubleshooting

### Plugin fails to load
- Ensure plugin is built with Go 1.24.2
- Ensure plugin implements required interface
- Check plugin exports `NewStrategy` function or `Strategy` variable

### Database connection fails
- Verify connection string format
- Ensure Neon database is accessible
- Check SSL mode is set to `require`

### WebSocket disconnects
- Check firewall settings
- Verify CORS configuration
- Monitor client send buffer size

## License

MIT

## Support

For issues and questions, please open an issue on GitHub.
