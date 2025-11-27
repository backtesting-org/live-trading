package performance

import (
	"strings"
	"sync"
	"time"
)

// ObjectPool provides generic object pooling for WebSocket message processing
type ObjectPool[T any] struct {
	pool      sync.Pool
	newFunc   func() T
	resetFunc func(T)
}

// NewObjectPool creates a new object pool with the given constructor and reset functions
func NewObjectPool[T any](newFunc func() T, resetFunc func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		newFunc:   newFunc,
		resetFunc: resetFunc,
	}
}

func (op *ObjectPool[T]) Get() T {
	return op.pool.Get().(T)
}

func (op *ObjectPool[T]) Put(obj T) {
	if op.resetFunc != nil {
		op.resetFunc(obj)
	}
	op.pool.Put(obj)
}

// Common pools for WebSocket message types
var (
	// ByteSlicePool for message buffers
	ByteSlicePool = NewObjectPool(
		func() []byte {
			return make([]byte, 0, 4096) // 4KB initial capacity
		},
		func(b []byte) {
			// Reset slice but keep capacity
			b = b[:0]
		},
	)

	// StringBuilderPool for message construction
	StringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	// MessagePool for generic message objects
	MessagePool = NewObjectPool(
		func() *GenericMessage {
			return &GenericMessage{}
		},
		func(msg *GenericMessage) {
			msg.Reset()
		},
	)
)

// GenericMessage represents a poolable message object
type GenericMessage struct {
	Type      string                 `json:"type"`
	Channel   string                 `json:"channel,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

func (gm *GenericMessage) Reset() {
	gm.Type = ""
	gm.Channel = ""
	gm.Symbol = ""
	if gm.Data != nil {
		for k := range gm.Data {
			delete(gm.Data, k)
		}
	}
	gm.Timestamp = time.Time{}
}

// GetByteSlice gets a byte slice from the pool
func GetByteSlice() []byte {
	return ByteSlicePool.Get()
}

// PutByteSlice returns a byte slice to the pool
func PutByteSlice(b []byte) {
	ByteSlicePool.Put(b)
}

// GetStringBuilder gets a string builder from the pool
func GetStringBuilder() *strings.Builder {
	return StringBuilderPool.Get().(*strings.Builder)
}

// PutStringBuilder returns a string builder to the pool
func PutStringBuilder(sb *strings.Builder) {
	sb.Reset()
	StringBuilderPool.Put(sb)
}

// GetMessage gets a generic message from the pool
func GetMessage() *GenericMessage {
	return MessagePool.Get()
}

// PutMessage returns a generic message to the pool
func PutMessage(msg *GenericMessage) {
	MessagePool.Put(msg)
}
