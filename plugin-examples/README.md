# Trading Strategy Plugins

Go runtime plugins for trading strategies using Kronos SDK.

## Overview

This directory contains plugin versions of trading strategies that can be dynamically loaded into the live-trading platform. Each strategy is compiled as a `.so` file and loaded at runtime.

## Strategies

### Grid Trading (`grid/`)
Market-neutral grid trading strategy that places buy/sell orders at regular price intervals.

**Features:**
- Configurable price range (upper/lower bounds)
- Adjustable grid step size
- Maximum concurrent orders limit
- Fixed quote-based order sizing

### Momentum (`momentum/`)
Kline-based momentum trading strategy that detects price trends.

**Features:**
- Configurable momentum thresholds
- Multiple asset support (BTC, ETH)
- Multi-exchange compatibility
- Adjustable kline intervals

## Requirements

- **Go 1.24.2** (exact version required for plugin compatibility)
- **Linux, macOS, or FreeBSD** (Go plugins supported on these platforms)
- **Kronos SDK**

## Building Plugins

### Build all plugins:
```bash
make all
```

### Build specific plugin:
```bash
make grid      # Build grid.so
make momentum  # Build momentum.so
```

### Test compilation (without building plugins):
```bash
make test
```

### Clean built plugins:
```bash
make clean
```

## Development

### Creating a New Plugin Strategy

1. **Create directory:**
```bash
mkdir strategy-plugins/my_strategy
```

2. **Implement strategy:**
```go
package main

import (
    "github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
    // ... other SDK imports
)

type MyStrategy struct {
    *strategy.BaseStrategy
    // ... your fields
}

func (s *MyStrategy) GetSignals() ([]*strategy.Signal, error) {
    // Your trading logic
}

// Required: Plugin export
var Strategy strategy.Strategy
```

3. **Add to Makefile:**
```makefile
my_strategy:
    cd my_strategy && go build -buildmode=plugin -o ../my_strategy.so .
```

4. **Build:**
```bash
make my_strategy
```

## Plugin Interface

All strategies must implement the `strategy.Strategy` interface from Kronos SDK:

```go
type Strategy interface {
    GetSignals() ([]*Signal, error)
    GetName() StrategyName
    GetDescription() string
    GetRiskLevel() RiskLevel
    GetStrategyType() StrategyType
    Enable() error
    Disable() error
    IsEnabled() bool
}
```

## Dependencies

Plugins use **only** Kronos SDK interfaces:
- `github.com/backtesting-org/kronos-sdk/pkg/types/strategy` - Strategy interfaces
- `github.com/backtesting-org/kronos-sdk/pkg/types/portfolio/store` - Market data access
- `github.com/backtesting-org/kronos-sdk/pkg/types/logging` - Logging
- `github.com/backtesting-org/kronos-sdk/pkg/types/connector` - Exchange types


## Important Notes

### Plugin Limitations

1. **Platform Support** - Go plugins work on Linux, macOS, and FreeBSD (not Windows)
2. **Go Version Must Match** - Plugin and main app must use identical Go version (1.24.2)
3. **Dependency Versions Must Match** - All shared dependencies must be identical
4. **Cannot Unload** - Go plugins cannot be unloaded once loaded (limitation of Go)

## Troubleshooting

### "plugin was built with a different version of package X"
- Ensure Go version matches exactly (1.24.2)
- Verify all dependency versions match between plugin and main app
- Rebuild with `make clean && make all`

### "plugin.Open: plugin.so: undefined symbol"
- Check that `var Strategy strategy.Strategy` is exported
- Ensure strategy implements all interface methods
- Verify build mode is `-buildmode=plugin`

### Plugins crash on load
- Check that all dependencies are compatible
- Verify no init() functions that could panic
- Test compilation with `make test` first
