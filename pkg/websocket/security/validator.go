package security

import (
	"encoding/json"
	"fmt"
)

type ValidationConfig struct {
	MaxMessageSize int
	AllowedTypes   map[string]bool
	RequiredFields map[string][]string
	TypeField      string // Field name for message type (default: "type", Hyperliquid: "channel")
}

type messageValidator struct {
	config ValidationConfig
}

func NewMessageValidator(config ValidationConfig) MessageValidator {
	return &messageValidator{config: config}
}

func (mv *messageValidator) ValidateMessage(message []byte) error {
	// Size validation
	if len(message) > mv.config.MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max: %d)",
			len(message), mv.config.MaxMessageSize)
	}

	// Parse base message
	var baseMsg map[string]interface{}
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Default TypeField to "type" if not specified
	typeField := mv.config.TypeField
	if typeField == "" {
		typeField = "type"
	}

	// Type validation - use configurable field name
	msgType, ok := baseMsg[typeField].(string)
	if !ok {
		return fmt.Errorf("missing or invalid message %s field", typeField)
	}

	if !mv.config.AllowedTypes[msgType] {
		return fmt.Errorf("invalid message type: %s", msgType)
	}

	// Required fields validation
	if requiredFields, exists := mv.config.RequiredFields[msgType]; exists {
		for _, field := range requiredFields {
			if _, exists := baseMsg[field]; !exists {
				return fmt.Errorf("missing required field '%s' for type '%s'", field, msgType)
			}
		}
	}

	return nil
}
