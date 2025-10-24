package base

import (
	"context"
	"encoding/json"
	"fmt"
)

// MessageHandler defines the interface for processing WebSocket messages
type MessageHandler interface {
	// Handle processes a raw WebSocket message
	Handle(ctx context.Context, message []byte) error

	// GetChannels returns the channels this handler is responsible for
	GetChannels() []string

	// GetMessageTypes returns the message types this handler can process
	GetMessageTypes() []string
}

// HandlerRegistry manages message handlers for different channels and types
type HandlerRegistry struct {
	handlers     map[string]MessageHandler // channel -> handler
	typeHandlers map[string]MessageHandler // message type -> handler
	logger       Logger
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

func NewHandlerRegistry(logger Logger) *HandlerRegistry {
	return &HandlerRegistry{
		handlers:     make(map[string]MessageHandler),
		typeHandlers: make(map[string]MessageHandler),
		logger:       logger,
	}
}

// RegisterHandler registers a handler for specific channels
func (hr *HandlerRegistry) RegisterHandler(handler MessageHandler) error {
	// Register by channels
	for _, channel := range handler.GetChannels() {
		if existing, exists := hr.handlers[channel]; exists {
			return fmt.Errorf("handler already registered for channel '%s': %T", channel, existing)
		}
		hr.handlers[channel] = handler
		hr.logger.Debug("Registered handler for channel: %s", channel)
	}

	// Register by message types
	for _, msgType := range handler.GetMessageTypes() {
		if existing, exists := hr.typeHandlers[msgType]; exists {
			return fmt.Errorf("handler already registered for message type '%s': %T", msgType, existing)
		}
		hr.typeHandlers[msgType] = handler
		hr.logger.Debug("Registered handler for message type: %s", msgType)
	}

	return nil
}

// RouteMessage routes a message to the appropriate handler
func (hr *HandlerRegistry) RouteMessage(ctx context.Context, message []byte) error {
	// Parse base message to determine routing
	var baseMsg BaseMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return fmt.Errorf("failed to parse base message: %w", err)
	}

	// Try routing by channel first
	if baseMsg.Channel != "" {
		if handler, exists := hr.handlers[baseMsg.Channel]; exists {
			return handler.Handle(ctx, message)
		}
	}

	// Fall back to routing by message type
	if baseMsg.Type != "" {
		if handler, exists := hr.typeHandlers[baseMsg.Type]; exists {
			return handler.Handle(ctx, message)
		}
	}

	hr.logger.Debug("No handler found for message - channel: %s, type: %s", baseMsg.Channel, baseMsg.Type)
	return nil
}

// GetRegisteredChannels returns all registered channels
func (hr *HandlerRegistry) GetRegisteredChannels() []string {
	channels := make([]string, 0, len(hr.handlers))
	for channel := range hr.handlers {
		channels = append(channels, channel)
	}
	return channels
}

// GetRegisteredTypes returns all registered message types
func (hr *HandlerRegistry) GetRegisteredTypes() []string {
	types := make([]string, 0, len(hr.typeHandlers))
	for msgType := range hr.typeHandlers {
		types = append(types, msgType)
	}
	return types
}

// BaseHandler provides common functionality for message handlers
type BaseHandler struct {
	channels     []string
	messageTypes []string
	logger       Logger
}

func NewBaseHandler(channels, messageTypes []string, logger Logger) *BaseHandler {
	return &BaseHandler{
		channels:     channels,
		messageTypes: messageTypes,
		logger:       logger,
	}
}

func (bh *BaseHandler) GetChannels() []string {
	return bh.channels
}

func (bh *BaseHandler) GetMessageTypes() []string {
	return bh.messageTypes
}

// ValidationHandler wraps another handler with validation
type ValidationHandler struct {
	wrapped   MessageHandler
	validator func([]byte) error
	logger    Logger
}

func NewValidationHandler(wrapped MessageHandler, validator func([]byte) error, logger Logger) *ValidationHandler {
	return &ValidationHandler{
		wrapped:   wrapped,
		validator: validator,
		logger:    logger,
	}
}

func (vh *ValidationHandler) Handle(ctx context.Context, message []byte) error {
	if vh.validator != nil {
		if err := vh.validator(message); err != nil {
			vh.logger.Warn("Message validation failed: %v", err)
			return fmt.Errorf("message validation failed: %w", err)
		}
	}

	return vh.wrapped.Handle(ctx, message)
}

func (vh *ValidationHandler) GetChannels() []string {
	return vh.wrapped.GetChannels()
}

func (vh *ValidationHandler) GetMessageTypes() []string {
	return vh.wrapped.GetMessageTypes()
}

// AsyncHandler wraps another handler to process messages asynchronously
type AsyncHandler struct {
	wrapped    MessageHandler
	workerPool chan struct{}
	logger     Logger
}

func NewAsyncHandler(wrapped MessageHandler, maxWorkers int, logger Logger) *AsyncHandler {
	return &AsyncHandler{
		wrapped:    wrapped,
		workerPool: make(chan struct{}, maxWorkers),
		logger:     logger,
	}
}

func (ah *AsyncHandler) Handle(ctx context.Context, message []byte) error {
	// Try to acquire a worker
	select {
	case ah.workerPool <- struct{}{}:
		// Got a worker, process asynchronously
		go func() {
			defer func() { <-ah.workerPool }()

			if err := ah.wrapped.Handle(ctx, message); err != nil {
				ah.logger.Error("Async handler error: %v", err)
			}
		}()
		return nil
	default:
		// No workers available, process synchronously
		ah.logger.Warn("No async workers available, processing synchronously")
		return ah.wrapped.Handle(ctx, message)
	}
}

func (ah *AsyncHandler) GetChannels() []string {
	return ah.wrapped.GetChannels()
}

func (ah *AsyncHandler) GetMessageTypes() []string {
	return ah.wrapped.GetMessageTypes()
}
