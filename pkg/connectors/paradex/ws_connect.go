package paradex

import (
	"fmt"
)

func (p *paradex) StartWebSocket() error {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	if p.wsService == nil {
		return fmt.Errorf("WebSocket service not initialized")
	}

	if p.wsService.IsConnected() {
		return nil
	}

	if err := p.wsService.Connect(); err != nil {
		return fmt.Errorf("failed to start paradex WebSocket: %w", err)
	}

	p.appLogger.Info("paradex WebSocket started successfully")
	return nil
}

func (p *paradex) StopWebSocket() error {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	if p.wsService == nil {
		return nil
	}

	if p.wsCancel != nil {
		p.wsCancel()
	}

	if err := p.wsService.Disconnect(); err != nil {
		return fmt.Errorf("failed to stop paradex WebSocket: %w", err)
	}

	p.appLogger.Info("paradex WebSocket stopped successfully")
	return nil
}

func (p *paradex) IsWebSocketConnected() bool {
	p.wsMutex.RLock()
	defer p.wsMutex.RUnlock()

	if p.wsService == nil {
		return false
	}

	return p.wsService.IsConnected()
}
