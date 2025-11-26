package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/backtesting-org/live-trading/internal/cli/arguments"
	"github.com/spf13/cobra"
)

// CreateMetadataCommand creates a cobra command that exports exchange metadata
func CreateMetadataCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list-exchanges",
		Short: "List all available exchanges and their configuration requirements",
		Long: `List all available exchanges and their configuration requirements.

By default, outputs in a human-readable format.
Use --json flag for machine-readable JSON output (useful for programmatic integration).`,
		Run: func(cmd *cobra.Command, args []string) {
			registry := arguments.NewRegistry()
			metadata := registry.GetAllMetadata()

			if jsonOutput {
				output, err := json.MarshalIndent(metadata, "", "  ")
				if err != nil {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Error marshalling metadata: %v\n", err)
					return
				}
				fmt.Println(string(output))
			} else {
				printHumanReadable(metadata)
			}
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	return cmd
}

func printHumanReadable(metadata []*arguments.ExchangeMetadata) {
	fmt.Println("Available Exchanges:")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, exchange := range metadata {
		if i > 0 {
			fmt.Println()
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println()
		}

		fmt.Printf("📊  %s\n", strings.ToUpper(exchange.Name))
		fmt.Println()

		// Separate required and optional flags
		var required, optional []arguments.FlagMetadata
		for _, flag := range exchange.Flags {
			if flag.Required {
				required = append(required, flag)
			} else {
				optional = append(optional, flag)
			}
		}

		// Print required flags
		if len(required) > 0 {
			fmt.Println("  Required Flags:")
			for _, flag := range required {
				printFlag(flag, true)
			}
		}

		// Print optional flags
		if len(optional) > 0 {
			if len(required) > 0 {
				fmt.Println()
			}
			fmt.Println("  Optional Flags:")
			for _, flag := range optional {
				printFlag(flag, false)
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

func printFlag(flag arguments.FlagMetadata, isRequired bool) {
	marker := "  "
	if isRequired {
		marker = "  * "
	} else {
		marker = "    "
	}

	fmt.Printf("%s--%-35s [%s]\n", marker, flag.Name, flag.Type)
	fmt.Printf("      %s\n", flag.Description)

	if flag.DefaultValue != nil && flag.DefaultValue != "" && flag.DefaultValue != false {
		fmt.Printf("      Default: %v\n", flag.DefaultValue)
	}
}
