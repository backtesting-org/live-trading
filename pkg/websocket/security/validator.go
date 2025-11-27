package security

import (
	"encoding/json"
	"fmt"
)

type ValidationConfig struct {
	MaxMessageSize int
	AllowedTypes   map[string]bool
	RequiredFields map[string][]string
}

type MessageValidator struct {
	config ValidationConfig
}

func NewMessageValidator(config ValidationConfig) *MessageValidator {
	return &MessageValidator{config: config}
}

func (mv *MessageValidator) ValidateMessage(message []byte) error {
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

	// Type validation
	msgType, ok := baseMsg["type"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid message type")
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
