package base

type BaseService interface {
	ProcessMessage(message []byte, handler func([]byte) error) error
	GetMetrics() map[string]interface{}
	IsConnected() bool
	SetConnected(connected bool)
}
