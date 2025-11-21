package handlers

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/backtesting-org/kronos-sdk/pkg/types/strategy"
)

// RunStrategy executes a strategy in a loop
func RunStrategy(strat strategy.Strategy) error {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	fmt.Printf("Running strategy: %s\n", strat.GetName())

	for {
		select {
		case <-ticker.C:
			signals, err := strat.GetSignals()
			if err != nil {
				fmt.Printf("Strategy error: %v\n", err)
				continue
			}
			fmt.Printf("Generated %d signals\n", len(signals))

		case <-sigChan:
			fmt.Println("Shutting down...")
			return nil
		}
	}
}
