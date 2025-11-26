package arguments

import (
	"fmt"

	"github.com/spf13/cobra"
)

// FlagType represents the type of a CLI flag
type FlagType string

const (
	FlagTypeString  FlagType = "string"
	FlagTypeBool    FlagType = "bool"
	FlagTypeFloat64 FlagType = "float64"
	FlagTypeInt     FlagType = "int"
)

// FlagMetadata describes a single CLI flag
type FlagMetadata struct {
	Name         string      `json:"name"`
	Type         FlagType    `json:"type"`
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
	EnvVar       string      `json:"env_var,omitempty"`
}

// ExchangeMetadata describes all metadata for an exchange
type ExchangeMetadata struct {
	Name  string         `json:"name"`
	Flags []FlagMetadata `json:"flags"`
}

// RegisterFlag registers a single flag with a cobra command based on its metadata
func (f *FlagMetadata) RegisterFlag(cmd *cobra.Command) error {
	switch f.Type {
	case FlagTypeString:
		defaultVal, _ := f.DefaultValue.(string)
		cmd.Flags().String(f.Name, defaultVal, f.Description)
	case FlagTypeBool:
		defaultVal, _ := f.DefaultValue.(bool)
		cmd.Flags().Bool(f.Name, defaultVal, f.Description)
	case FlagTypeFloat64:
		defaultVal, _ := f.DefaultValue.(float64)
		cmd.Flags().Float64(f.Name, defaultVal, f.Description)
	case FlagTypeInt:
		defaultVal, _ := f.DefaultValue.(int)
		cmd.Flags().Int(f.Name, defaultVal, f.Description)
	default:
		return fmt.Errorf("unsupported flag type: %s", f.Type)
	}
	return nil
}

// GetValue retrieves the value of this flag from a cobra command
func (f *FlagMetadata) GetValue(cmd *cobra.Command) (interface{}, error) {
	switch f.Type {
	case FlagTypeString:
		return cmd.Flags().GetString(f.Name)
	case FlagTypeBool:
		return cmd.Flags().GetBool(f.Name)
	case FlagTypeFloat64:
		return cmd.Flags().GetFloat64(f.Name)
	case FlagTypeInt:
		return cmd.Flags().GetInt(f.Name)
	default:
		return nil, fmt.Errorf("unsupported flag type: %s", f.Type)
	}
}

// RegisterFlags registers all flags for this exchange with a cobra command
func (e *ExchangeMetadata) RegisterFlags(cmd *cobra.Command) error {
	for _, flag := range e.Flags {
		if err := flag.RegisterFlag(cmd); err != nil {
			return fmt.Errorf("failed to register flag %s for %s: %w", flag.Name, e.Name, err)
		}
	}
	return nil
}

// GetFlagValues retrieves all flag values from a cobra command
func (e *ExchangeMetadata) GetFlagValues(cmd *cobra.Command) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for _, flag := range e.Flags {
		val, err := flag.GetValue(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to get value for flag %s: %w", flag.Name, err)
		}
		values[flag.Name] = val
	}
	return values, nil
}
