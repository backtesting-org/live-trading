package services

import (
	"sync"
)

// EventType represents the type of event
type EventType string

const (
	EventStrategyStarted   EventType = "strategy_started"
	EventStrategyStopped   EventType = "strategy_stopped"
	EventStrategyError     EventType = "strategy_error"
	EventSignalGenerated   EventType = "signal_generated"
	EventOrderPlaced       EventType = "order_placed"
	EventOrderFilled       EventType = "order_filled"
	EventTradeExecuted     EventType = "trade_executed"
	EventAccountUpdate     EventType = "account_update"
)

// Event represents a system event
type Event struct {
	Type EventType              `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// EventBus manages event subscriptions and publishing
type EventBus struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[EventType][]chan Event),
	}
}

// Subscribe creates a subscription to events of a specific type
func (eb *EventBus) Subscribe(eventType EventType, bufferSize int) <-chan Event {
	ch := make(chan Event, bufferSize)

	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	return ch
}

// SubscribeAll creates a subscription to all event types
func (eb *EventBus) SubscribeAll(bufferSize int) <-chan Event {
	ch := make(chan Event, bufferSize)

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Subscribe to all known event types
	allEventTypes := []EventType{
		EventStrategyStarted,
		EventStrategyStopped,
		EventStrategyError,
		EventSignalGenerated,
		EventOrderPlaced,
		EventOrderFilled,
		EventTradeExecuted,
		EventAccountUpdate,
	}

	for _, eventType := range allEventTypes {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
	}

	return ch
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	subscribers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	// Send event to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip this subscriber
		}
	}
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(eventType EventType, ch <-chan Event) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers := eb.subscribers[eventType]
	for i, subscriber := range subscribers {
		if subscriber == ch {
			// Remove from slice
			eb.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			close(subscriber)
			break
		}
	}
}

// Close closes all subscriber channels
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for eventType, subscribers := range eb.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(eb.subscribers, eventType)
	}
}
