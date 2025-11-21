package cli

import (
	"fmt"
	"os"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/registry"
	"github.com/backtesting-org/live-trading/internal/cli/arguments"
	"github.com/backtesting-org/live-trading/internal/cli/handlers"
	pkgconnector "github.com/backtesting-org/live-trading/pkg/connector"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// Module provides the CLI commands
var Module = fx.Module("cli",
	fx.Provide(
		arguments.NewRegistry,
		NewRunCmd,
	),
	fx.Invoke(RunCLI),
)

// NewRunCmd creates the run command
func NewRunCmd(argRegistry *arguments.Registry) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a trading strategy",
	}

	cmd.Flags().StringP("exchange", "e", "", "Exchange to connect to (paradex, bybit, etc.)")
	cmd.Flags().StringP("strategy", "s", "", "Path to strategy plugin (.so file)")
	cmd.MarkFlagRequired("exchange")
	cmd.MarkFlagRequired("strategy")

	// Register all exchange-specific flags
	argRegistry.RegisterAllFlags(cmd)

	return cmd
}

// RunCLI executes the cobra CLI with fx dependencies
func RunCLI(runCmd *cobra.Command, registry registry.ConnectorRegistry, argRegistry *arguments.Registry) {
	rootCmd := &cobra.Command{
		Use:   "kronos-live",
		Short: "Kronos live trading CLI",
	}

	rootCmd.AddCommand(runCmd)

	// Set the run command's RunE function with injected dependencies
	runCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runStrategy(cmd, registry, argRegistry)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runStrategy executes the strategy with access to the registry
func runStrategy(cmd *cobra.Command, registry registry.ConnectorRegistry, argRegistry *arguments.Registry) error {
	exchange, _ := cmd.Flags().GetString("exchange")
	strategyPath, _ := cmd.Flags().GetString("strategy")

	fmt.Printf("Running strategy: %s on exchange: %s\n", strategyPath, exchange)

	// Get argument handler for this exchange
	argHandler, err := argRegistry.GetHandler(exchange)
	if err != nil {
		return err
	}

	// Parse config from flags
	config, err := argHandler.ParseConfig(cmd)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Convert string to ExchangeName
	var exchangeName connector.ExchangeName
	switch exchange {
	case "paradex":
		exchangeName = connector.Paradex
	case "bybit":
		exchangeName = connector.Bybit
	default:
		return fmt.Errorf("unsupported exchange: %s", exchange)
	}

	// Get connector from registry
	conn, err := handlers.GetConnectorFromRegistry(registry, exchangeName)
	if err != nil {
		return fmt.Errorf("failed to get connector: %w", err)
	}

	fmt.Printf("Successfully got connector: %T\n", conn)

	// Initialize connector with config
	initConn, ok := conn.(pkgconnector.Initializable)
	if !ok {
		return fmt.Errorf("connector %s does not support initialization", exchange)
	}

	if err := initConn.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize connector: %w", err)
	}

	fmt.Printf("Connector initialized successfully\n")

	// TODO: Initialize Kronos SDK
	// TODO: Load plugin
	// TODO: Run strategy

	return fmt.Errorf("implementation incomplete - need kronos initialization and plugin loading")
}
