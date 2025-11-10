package paradex

import (
	"context"
	"fmt"
)

func (p *Paradex) StartWebSocket(ctx context.Context) error {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	if p.wsService == nil {
		return fmt.Errorf("WebSocket service not initialized")
	}

	if p.wsService.IsConnected() {
		return nil
	}

	p.wsContext, p.wsCancel = context.WithCancel(ctx)

	if err := p.wsService.Connect(p.wsContext); err != nil {
		return fmt.Errorf("failed to start Paradex WebSocket: %w", err)
	}

	p.appLogger.Info("Paradex WebSocket started successfully")
	return nil
}

func (p *Paradex) StopWebSocket() error {
	p.wsMutex.Lock()
	defer p.wsMutex.Unlock()

	if p.wsService == nil {
		return nil
	}

	if p.wsCancel != nil {
		p.wsCancel()
	}

	if err := p.wsService.Disconnect(); err != nil {
		return fmt.Errorf("failed to stop Paradex WebSocket: %w", err)
	}

	p.appLogger.Info("Paradex WebSocket stopped successfully")
	return nil
}

func (p *Paradex) IsWebSocketConnected() bool {
	p.wsMutex.RLock()
	defer p.wsMutex.RUnlock()

	if p.wsService == nil {
		return false
	}

	return p.wsService.IsConnected()
}
