package cli

import (
	"fmt"
	"os"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/internal/cli/handlers"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// Module provides the CLI commands
var Module = fx.Module("cli",
	fx.Provide(
		NewRunCmd,
	),
	fx.Invoke(RunCLI),
)

// NewRunCmd creates the run command
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a trading strategy",
	}

	cmd.Flags().StringP("exchange", "e", "", "Exchange to connect to (paradex, bybit, etc.)")
	cmd.Flags().StringP("strategy", "s", "", "Path to strategy plugin (.so file)")
	cmd.MarkFlagRequired("exchange")
	cmd.MarkFlagRequired("strategy")

	return cmd
}

// RunCLI executes the cobra CLI with fx dependencies
func RunCLI(runCmd *cobra.Command, registry registry.ConnectorRegistry) {
	rootCmd := &cobra.Command{
		Use:   "kronos-live",
		Short: "Kronos live trading CLI",
	}

	rootCmd.AddCommand(runCmd)

	// Set the run command's RunE function with injected dependencies
	runCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runStrategy(cmd, registry)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runStrategy executes the strategy with access to the registry
func runStrategy(cmd *cobra.Command, registry registry.ConnectorRegistry) error {
	exchange, _ := cmd.Flags().GetString("exchange")
	strategyPath, _ := cmd.Flags().GetString("strategy")

	fmt.Printf("Running strategy: %s on exchange: %s\n", strategyPath, exchange)

	// Get connector from registry
	conn, err := handlers.GetConnectorFromRegistry(registry, connector.ExchangeName(exchange))
	if err != nil {
		return fmt.Errorf("failed to get connector: %w", err)
	}

	fmt.Printf("Successfully got connector: %T\n", conn.GetConnectorInfo())

	// TODO: Initialize Kronos SDK
	// TODO: Load plugin
	// TODO: Run strategy

	return fmt.Errorf("implementation incomplete - need kronos initialization and plugin loading")
}
