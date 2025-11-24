package real_time

import (
	"context"

	"github.com/sonirico/go-hyperliquid"
)

type RealTimeData struct {
	ws *hyperliquid.WebsocketClient
}

func NewRealTimeDataService(ws *hyperliquid.WebsocketClient) *RealTimeData {
	return &RealTimeData{ws: ws}
}

func (w *RealTimeData) Connect(ctx context.Context) error {
	return w.ws.Connect(ctx)
}

func (w *RealTimeData) Disconnect() error {
	return w.ws.Close()
}
