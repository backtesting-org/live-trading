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
	),
)

// CLIArgs holds parsed command-line arguments
type CLIArgs struct {
	Exchange     string
	StrategyPath string
	RawArgs      []string
}

// ParseCLI parses command-line arguments without initializing fx
// Returns nil if help was requested or command doesn't need fx
func ParseCLI() *CLIArgs {
	var cliArgs *CLIArgs

	rootCmd := &cobra.Command{
		Use:   "kronos-live",
		Short: "Kronos live trading CLI",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a trading strategy",
		Long: `Run a trading strategy on a live exchange.

To see exchange-specific options, specify the exchange:
  kronos-live run --exchange paradex --help

Examples:
  # Run with Paradex
  kronos-live run --exchange paradex --strategy ./strategy.so \
    --paradex-account-address 0x... --paradex-eth-private-key 0x...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			exchange, _ := cmd.Flags().GetString("exchange")
			strategyPath, _ := cmd.Flags().GetString("strategy")

			if exchange == "" {
				return fmt.Errorf("--exchange is required")
			}
			if strategyPath == "" {
				return fmt.Errorf("--strategy is required")
			}

			// Store parsed args for fx execution
			cliArgs = &CLIArgs{
				Exchange:     exchange,
				StrategyPath: strategyPath,
				RawArgs:      os.Args,
			}
			return nil
		},
	}

	runCmd.Flags().StringP("exchange", "e", "", "Exchange to connect to (paradex, bybit, etc.)")
	runCmd.Flags().StringP("strategy", "s", "", "Path to strategy plugin (.so file)")

	// Check if user wants help for a specific exchange
	for i, arg := range os.Args {
		if arg == "--exchange" || arg == "-e" {
			if i+1 < len(os.Args) {
				exchange := os.Args[i+1]
				// Add exchange-specific flags for help
				registry := arguments.NewRegistry()
				if handler, err := registry.GetHandler(exchange); err == nil {
					handler.RegisterFlags(runCmd)
				}
				break
			}
		}
	}

	rootCmd.AddCommand(runCmd)

	// Execute cobra - will return early if --help is used
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cliArgs
}

// ExecuteStrategy runs the trading strategy with fx dependencies
func ExecuteStrategy(cliArgs *CLIArgs, registry registry.ConnectorRegistry, argRegistry *arguments.Registry) error {
	fmt.Printf("Running strategy: %s on exchange: %s\n", cliArgs.StrategyPath, cliArgs.Exchange)

	// Recreate cobra command to parse exchange-specific flags
	cmd := &cobra.Command{Use: "run"}
	cmd.Flags().StringP("exchange", "e", "", "")
	cmd.Flags().StringP("strategy", "s", "", "")

	// Get argument handler and register its flags
	argHandler, err := argRegistry.GetHandler(cliArgs.Exchange)
	if err != nil {
		return err
	}
	argHandler.RegisterFlags(cmd)

	// Parse the original args
	cmd.SetArgs(cliArgs.RawArgs[2:]) // Skip "kronos-live run"
	if err := cmd.ParseFlags(cliArgs.RawArgs[2:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Parse config from flags
	config, err := argHandler.ParseConfig(cmd)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Convert string to ExchangeName
	var exchangeName connector.ExchangeName
	switch cliArgs.Exchange {
	case "paradex":
		exchangeName = connector.Paradex
	case "bybit":
		exchangeName = connector.Bybit
	case "hyperliquid":
		exchangeName = connector.Hyperliquid
	default:
		return fmt.Errorf("unsupported exchange: %s", cliArgs.Exchange)
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
		return fmt.Errorf("connector %s does not support initialization", cliArgs.Exchange)
	}

	if err := initConn.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize connector: %w", err)
	}

	fmt.Printf("Connector initialized successfully\n")

	return fmt.Errorf("implementation incomplete - need kronos initialization and plugin loading")
}
