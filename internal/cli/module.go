package cli

import (
	"fmt"
	"os"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/logging"
	"github.com/backtesting-org/kronos-sdk/pkg/types/plugin"
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
				argRegistry := arguments.NewRegistry()
				if handler, err := argRegistry.GetHandler(exchange); err == nil {
					handler.RegisterFlags(runCmd)
				}
				break
			}
		}
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(CreateMetadataCommand())

	// Execute cobra - will return early if --help is used
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return cliArgs
}

// ExecuteStrategy runs the trading strategy with fx dependencies
func ExecuteStrategy(
	cliArgs *CLIArgs,
	registry registry.ConnectorRegistry,
	argRegistry *arguments.Registry,
	pluginManager plugin.Manager,
	logger logging.ApplicationLogger,
) error {
	logger.Info("Starting strategy execution", "strategyPath", cliArgs.StrategyPath)

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

	logger.Info("Connecting to exchange", "exchange", exchangeName)

	// Initialize connector with config
	initConn, ok := conn.(pkgconnector.Initializable)
	if !ok {
		logger.Error(fmt.Sprintf("Connector %s does not support initialization", cliArgs.Exchange))
		return fmt.Errorf("connector %s does not support initialization", cliArgs.Exchange)
	}

	if err := initConn.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize connector: %w", err)
	}

	logger.Info("Successfully connected to exchange", "exchange", exchangeName)

	loadPlugin, err := pluginManager.LoadStrategyPlugin(cliArgs.StrategyPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to load plugin: %s", err))
		return err
	}

	logger.Info("Successfully loaded strategy plugin", "strategy", loadPlugin.GetName())

	return nil
}
