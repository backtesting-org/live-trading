# Live Trading Codebase Analysis: Library vs CLI Separation

## Executive Summary

After investigating the codebase, here's what should stay in a **library** vs what should be **CLI-only**:

---

## ✅ KEEP IN LIBRARY (Core Trading Logic - 90%)

### 1. **Exchange Connectors** (`external/connectors/`)
**Purpose:** Core trading logic, market data, order execution
**Keep:** ✅ All of it

```
external/connectors/
├── bybit/
├── hyperliquid/
├── paradex/
└── module.go
```

**Why:** This is the heart of live trading. Any system that wants to trade needs these connectors.

**Becomes:** `pkg/livetrading/connectors/` in the library

---

### 2. **Connector Interface** (`pkg/connector/`)
**Purpose:** Defines contract for all exchange implementations
**Keep:** ✅ All of it

```
pkg/connector/
├── config.go          # Config interface
└── initializable.go   # Initializable interface
```

**Why:** Shared interface needed by both CLI and any other consumers of the library.

**Becomes:** `pkg/livetrading/connector/` in the library

---

### 3. **WebSocket Infrastructure** (`external/websocket/`)
**Purpose:** Reusable WebSocket connection management
**Keep:** ✅ All of it

```
external/websocket/
├── base/              # Base WebSocket service
├── connection/        # Connection management
├── performance/       # Circuit breakers, metrics
└── security/          # Rate limiting, auth
```

**Why:** Shared by all exchange connectors. Critical for real-time data.

**Becomes:** `pkg/livetrading/websocket/` in the library

---

### 4. **Trading Core Logic** (Implied, needs to be extracted)
**Purpose:** Strategy execution, order management, risk controls
**Keep:** ✅ Create as library

Currently embedded in `cmd/live/main.go` via fx.New() - this needs to become:

```go
// pkg/livetrading/engine.go
type Engine struct {
    connectors map[string]connector.Initializable
    strategy   strategy.Strategy
    logger     logging.ApplicationLogger
}

func NewEngine(config *Config) *Engine { ... }
func (e *Engine) Run() error { ... }
```

**Why:** The actual "run a strategy against exchanges" logic should be importable.

---

## ❌ MOVE OUT TO CLI (Interface Layer - 10%)

### 1. **CLI Argument Parsing** (`internal/cli/`)
**Purpose:** Cobra commands, flag definitions, TUI
**Move:** ❌ CLI-only

```
internal/cli/
├── arguments/         # Exchange-specific flag parsing
├── handlers/          # CLI command handlers
├── metadata_command.go
└── module.go
```

**Why:** This is pure CLI interface code. Library consumers don't need Cobra.

**Becomes:** `cmd/kronos/internal/cli/` in the main CLI repo

---

### 2. **Main Entrypoint** (`cmd/live/`)
**Purpose:** Binary entrypoint with fx dependency injection
**Move:** ❌ CLI-only

```
cmd/live/main.go
```

**Why:** Library consumers will create their own entrypoints.

**Becomes:** `cmd/kronos-live/main.go` (thin wrapper calling the library)

---

### 3. **Database Repository** (`internal/database/`)
**Purpose:** PostgreSQL storage for strategy runs, signals
**Move:** ❌ Optional - depends on use case

```
internal/database/
├── migrations/
├── models.go
├── module.go
└── repository.go
```

**Why:** This is specific to the API server use case (which doesn't exist yet). If you're building a library for live trading, consumers might have their own storage.

**Options:**
- **Option A:** Keep as `pkg/livetrading/storage/` (optional dependency)
- **Option B:** Remove entirely (CLI handles persistence if needed)

**Recommendation:** Remove for now. Add back later if needed.

---

### 4. **Environment-Based Config** (`config/exchanges/`)
**Purpose:** Load exchange configs from env vars
**Move:** ❌ CLI-only

```
config/exchanges/
├── bybit.go
├── hyperliquid.go
├── paradex.go
└── module.go
```

**Why:** Library consumers will pass configs programmatically. Env var loading is CLI-specific.

**Becomes:** `cmd/kronos/internal/config/` in the main CLI repo

---

## 🤔 BORDERLINE CASES

### Plugin Examples (`plugin-examples/`)
**Current:** Example strategy implementations
**Decision:** ❌ Remove from library

**Why:** These are examples for users. They belong in documentation or a separate examples repo, not in the core library.

**Becomes:** Part of `kronos` CLI documentation or a separate `kronos-examples` repo

---

## 📊 Migration Path: Current → Library

### Current Structure
```
live-trading/
├── cmd/live/                    # ❌ CLI-only
├── config/exchanges/            # ❌ CLI-only
├── internal/
│   ├── cli/                     # ❌ CLI-only
│   └── database/                # ❌ Remove (not needed)
├── external/
│   ├── connectors/              # ✅ LIBRARY
│   └── websocket/               # ✅ LIBRARY
├── pkg/connector/               # ✅ LIBRARY
└── plugin-examples/             # ❌ Move to docs
```

### Proposed Library Structure
```
kronos/
└── pkg/
    └── livetrading/                    # THE LIBRARY
        ├── engine.go                   # NEW: Core execution engine
        ├── connector/                  # From pkg/connector/
        │   ├── config.go
        │   └── initializable.go
        ├── connectors/                 # From external/connectors/
        │   ├── bybit/
        │   ├── hyperliquid/
        │   ├── paradex/
        │   └── module.go
        └── websocket/                  # From external/websocket/
            ├── base/
            ├── connection/
            ├── performance/
            └── security/
```

### Proposed CLI Structure
```
kronos/
└── cmd/
    └── kronos-live/
        ├── main.go                     # Thin wrapper
        └── internal/
            ├── cli/                    # From internal/cli/
            └── config/                 # From config/exchanges/
```

---

## 🔧 Required Changes

### 1. Extract Engine Logic
Currently the trading loop is embedded in `cmd/live/main.go` using fx. Need to extract:

```go
// Before (cmd/live/main.go)
fx.New(
    kronos.Module,
    connectors.Module,
    cli.Module,
    fx.Invoke(cli.ExecuteStrategy),
).Run()

// After (pkg/livetrading/engine.go)
type Engine struct { ... }
func NewEngine(...) *Engine { ... }
func (e *Engine) LoadStrategy(path string) error { ... }
func (e *Engine) AddExchange(cfg connector.Config) error { ... }
func (e *Engine) Run(ctx context.Context) error { ... }
```

Then CLI becomes:
```go
// cmd/kronos-live/main.go
import "github.com/backtesting-org/kronos/pkg/livetrading"

func main() {
    engine := livetrading.NewEngine(...)
    engine.LoadStrategy(strategyPath)
    engine.AddExchange(hyperliquidConfig)
    engine.Run(context.Background())
}
```

### 2. Remove fx Dependency from Library
The library should NOT require uber/fx. That's an implementation detail of the CLI.

### 3. Make Connectors Standalone
Each connector should be usable independently:

```go
// Library consumer can use connectors directly
import "github.com/backtesting-org/kronos/pkg/livetrading/connectors/hyperliquid"

conn := hyperliquid.New(...)
conn.Initialize(config)
price := conn.FetchCurrentPrice("ETH-USD")
```

---

## 📦 What Gets Published

### Library: `github.com/backtesting-org/kronos`
```
go get github.com/backtesting-org/kronos/pkg/livetrading
```

Users can:
```go
import "github.com/backtesting-org/kronos/pkg/livetrading"

engine := livetrading.NewEngine(...)
engine.Run()
```

### CLI: Same repo, different binary
```
brew install kronos
kronos live --strategy ./my.so --exchange hyperliquid ...
```

---

## 🎯 Summary

| Component | Keep in Library? | Reason |
|-----------|------------------|--------|
| **Connectors** (`external/connectors/`) | ✅ YES | Core trading logic |
| **WebSocket** (`external/websocket/`) | ✅ YES | Reusable infrastructure |
| **Connector Interface** (`pkg/connector/`) | ✅ YES | Shared contract |
| **Engine** (needs extraction) | ✅ YES | Strategy execution |
| **CLI Args** (`internal/cli/`) | ❌ NO | Interface layer |
| **Config Loading** (`config/exchanges/`) | ❌ NO | CLI-specific |
| **Database** (`internal/database/`) | ❌ NO | Not needed yet |
| **Plugin Examples** | ❌ NO | Documentation |
| **Main Entrypoint** (`cmd/live/`) | ❌ NO | Thin wrapper |

**Core principle:** If it's about *how to trade*, keep it in the library. If it's about *how to interact with users*, move it to CLI.

