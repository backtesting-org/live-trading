package clients

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonirico/go-hyperliquid"
)

// ExchangeClient interface for trading operations with lazy configuration
type ExchangeClient interface {
	Configure(baseURL, privateKey, vaultAddr, accountAddr string) error
	IsConfigured() bool
	GetExchange() (*hyperliquid.Exchange, error)
}

// InfoClient interface for market data queries with lazy configuration
type InfoClient interface {
	Configure(baseURL string) error
	IsConfigured() bool
	GetInfo() (*hyperliquid.Info, error)
}

// WebSocketClient interface for real-time data with lazy configuration
type WebSocketClient interface {
	Configure(baseURL, privateKey string) error
	IsConfigured() bool
	GetWebSocket() (*hyperliquid.WebsocketClient, error)
}

// exchangeClient implementation
type exchangeClient struct {
	exchange   *hyperliquid.Exchange
	configured bool
	mu         sync.RWMutex
}

// infoClient implementation
type infoClient struct {
	info       *hyperliquid.Info
	configured bool
	mu         sync.RWMutex
}

// webSocketClient implementation
type webSocketClient struct {
	ws         *hyperliquid.WebsocketClient
	configured bool
	mu         sync.RWMutex
}

// NewExchangeClient creates an unconfigured exchange client
func NewExchangeClient() ExchangeClient {
	return &exchangeClient{
		configured: false,
	}
}

// NewInfoClient creates an unconfigured info client
func NewInfoClient() InfoClient {
	return &infoClient{
		configured: false,
	}
}

// Configure sets up the exchange client with runtime config
func (e *exchangeClient) Configure(baseURL, privateKey, vaultAddr, accountAddr string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.configured {
		return fmt.Errorf("client already configured")
	}

	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	e.exchange = hyperliquid.NewExchange(
		privateKeyECDSA,
		baseURL,
		nil, // Meta will be fetched automatically
		vaultAddr,
		accountAddr,
		nil, // SpotMeta will be fetched automatically
	)
	e.configured = true
	return nil
}

func (e *exchangeClient) IsConfigured() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.configured
}

func (e *exchangeClient) GetExchange() (*hyperliquid.Exchange, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.configured {
		return nil, fmt.Errorf("exchange client not configured")
	}
	return e.exchange, nil
}

// Configure sets up the info client with runtime config
func (i *infoClient) Configure(baseURL string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.configured {
		return fmt.Errorf("client already configured")
	}

	i.info = hyperliquid.NewInfo(baseURL, true, nil, nil)
	i.configured = true
	return nil
}

func (i *infoClient) IsConfigured() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.configured
}

func (i *infoClient) GetInfo() (*hyperliquid.Info, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if !i.configured {
		return nil, fmt.Errorf("info client not configured")
	}
	return i.info, nil
}

// NewWebSocketClient creates an unconfigured websocket client
func NewWebSocketClient() WebSocketClient {
	return &webSocketClient{
		configured: false,
	}
}

// Configure sets up the websocket client with runtime config
func (w *webSocketClient) Configure(baseURL, privateKey string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.configured {
		return fmt.Errorf("client already configured")
	}

	w.ws = hyperliquid.NewWebsocketClient(baseURL)
	w.configured = true
	return nil
}

func (w *webSocketClient) IsConfigured() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.configured
}

func (w *webSocketClient) GetWebSocket() (*hyperliquid.WebsocketClient, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if !w.configured {
		return nil, fmt.Errorf("websocket client not configured")
	}
	return w.ws, nil
}
