package handlers

import (
	"fmt"
	"plugin"

	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
)

// LoadStrategyPlugin loads a strategy plugin
func LoadStrategyPlugin(path string, kronos interface{}) (strategy.Strategy, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	sym, err := p.Lookup("Strategy")
	if err != nil {
		return nil, fmt.Errorf("failed to find Strategy symbol: %w", err)
	}

	// Type assert and initialize with Kronos
	strat := sym.(func(interface{}) strategy.Strategy)(kronos)
	return strat, nil
}
