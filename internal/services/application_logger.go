package services

import (
	"fmt"

	"go.uber.org/zap"
)

// ApplicationLoggerAdapter wraps zap.Logger to implement the ApplicationLogger interface
type ApplicationLoggerAdapter struct {
	logger *zap.Logger
}

// NewApplicationLogger creates a new application logger adapter
func NewApplicationLogger(logger *zap.Logger) *ApplicationLoggerAdapter {
	return &ApplicationLoggerAdapter{
		logger: logger,
	}
}

// Info logs an info message
func (al *ApplicationLoggerAdapter) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Info(msg)
}

// Debug logs a debug message
func (al *ApplicationLoggerAdapter) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Debug(msg)
}

// Warn logs a warning message
func (al *ApplicationLoggerAdapter) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Warn(msg)
}

// Error logs an error message
func (al *ApplicationLoggerAdapter) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Error(msg)
}

// Fatal logs a fatal message and exits
func (al *ApplicationLoggerAdapter) Fatal(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Fatal(msg)
}

// ErrorWithDebug logs an error with raw response data
func (al *ApplicationLoggerAdapter) ErrorWithDebug(msg string, rawResponse []byte, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	al.logger.Error(msg, zap.ByteString("raw_response", rawResponse))
}
