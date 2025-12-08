package connection

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConn abstracts the gorilla/websocket.Conn for testability
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// WebSocketDialer abstracts websocket dialing for testability
type WebSocketDialer interface {
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (WebSocketConn, *http.Response, error)
}

// gorillaWebSocketConn adapts gorilla/websocket.Conn to our interface
type gorillaWebSocketConn struct {
	conn *websocket.Conn
}

func (g *gorillaWebSocketConn) ReadMessage() (int, []byte, error) {
	return g.conn.ReadMessage()
}

func (g *gorillaWebSocketConn) WriteMessage(messageType int, data []byte) error {
	return g.conn.WriteMessage(messageType, data)
}

func (g *gorillaWebSocketConn) Close() error {
	return g.conn.Close()
}

func (g *gorillaWebSocketConn) SetReadDeadline(t time.Time) error {
	return g.conn.SetReadDeadline(t)
}

func (g *gorillaWebSocketConn) SetWriteDeadline(t time.Time) error {
	return g.conn.SetWriteDeadline(t)
}

// gorillaWebSocketDialer adapts gorilla/websocket.Dialer to our interface
type gorillaWebSocketDialer struct {
	dialer *websocket.Dialer
}

// NewGorillaDialer creates a production WebSocket dialer using gorilla/websocket
func NewGorillaDialer(config Config) WebSocketDialer {
	return &gorillaWebSocketDialer{
		dialer: &websocket.Dialer{
			HandshakeTimeout: config.HandshakeTimeout,
			ReadBufferSize:   config.ReadBufferSize,
			WriteBufferSize:  config.WriteBufferSize,
		},
	}
}

func (g *gorillaWebSocketDialer) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (WebSocketConn, *http.Response, error) {
	conn, resp, err := g.dialer.DialContext(ctx, urlStr, requestHeader)
	if err != nil {
		return nil, resp, err
	}

	// Wrap gorilla conn in our adapter
	return &gorillaWebSocketConn{conn: conn}, resp, nil
}
