package performance

import (
	"sync"
	"time"
)

type Metrics struct {
	MessagesReceived  int64
	MessagesProcessed int64
	MessagesDropped   int64
	ConnectionErrors  int64
	ReconnectionCount int64
	LastMessageTime   time.Time
	ProcessingLatency time.Duration
	mutex             sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncrementReceived() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MessagesReceived++
	m.LastMessageTime = time.Now()
}

func (m *Metrics) IncrementProcessed(latency time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MessagesProcessed++
	m.ProcessingLatency = latency
}

func (m *Metrics) IncrementDropped() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.MessagesDropped++
}

func (m *Metrics) IncrementConnectionError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ConnectionErrors++
}

func (m *Metrics) IncrementReconnection() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ReconnectionCount++
}

func (m *Metrics) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return map[string]interface{}{
		"messages_received":     m.MessagesReceived,
		"messages_processed":    m.MessagesProcessed,
		"messages_dropped":      m.MessagesDropped,
		"connection_errors":     m.ConnectionErrors,
		"reconnection_count":    m.ReconnectionCount,
		"last_message_time":     m.LastMessageTime,
		"processing_latency_ms": m.ProcessingLatency.Milliseconds(),
	}
}
