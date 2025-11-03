package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/backtesting-org/live-trading/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins - configure CORS properly in production
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	eventBus *services.EventBus
	logger   *zap.Logger
	clients  map[*Client]bool
	mu       sync.RWMutex
}

// Client represents a connected WebSocket client
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	handler   *Handler
	logger    *zap.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(eventBus *services.EventBus, logger *zap.Logger) *Handler {
	return &Handler{
		eventBus: eventBus,
		logger:   logger,
		clients:  make(map[*Client]bool),
	}
}

// HandleConnection handles a new WebSocket connection
// GET /ws
func (h *Handler) HandleConnection(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Create new client
	client := &Client{
		conn:    conn,
		send:    make(chan []byte, 256),
		handler: h,
		logger:  h.logger,
	}

	// Register client
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	h.logger.Info("New WebSocket client connected",
		zap.String("remote_addr", conn.RemoteAddr().String()))

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// BroadcastEvent broadcasts an event to all connected clients
func (h *Handler) BroadcastEvent(event services.Event) {
	message, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal event", zap.Error(err))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			// Client send buffer is full, close connection
			h.logger.Warn("Client send buffer full, closing connection")
			go h.unregisterClient(client)
		}
	}
}

// StartEventListener starts listening for events and broadcasting them
func (h *Handler) StartEventListener() {
	// Subscribe to all events
	eventChan := h.eventBus.SubscribeAll(100)

	go func() {
		for event := range eventChan {
			h.BroadcastEvent(event)
		}
	}()
}

// unregisterClient removes a client from the handler
func (h *Handler) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		client.conn.Close()
		h.logger.Info("WebSocket client disconnected")
	}
}

// GetClientCount returns the number of connected clients
func (h *Handler) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// readPump pumps messages from the WebSocket connection to the handler
func (c *Client) readPump() {
	defer func() {
		c.handler.unregisterClient(c)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// Log received message (currently we don't process client messages)
		c.logger.Debug("Received message from client", zap.ByteString("message", message))
	}
}

// writePump pumps messages from the handler to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The handler closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
