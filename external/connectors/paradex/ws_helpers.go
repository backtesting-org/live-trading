package paradex

import (
	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	websockets "github.com/backtesting-org/live-trading/external/connectors/paradex/websocket"
	padexmodel "github.com/trishtzy/go-paradex/models"

	"github.com/shopspring/decimal"
	"strings"
)

func (p *Paradex) convertOrderBookUpdates(output chan<- connector.OrderBook) {
	defer close(output)

	for wsUpdate := range p.wsService.OrderbookUpdates() {
		asset := p.parseAssetFromSymbol(wsUpdate.Symbol)

		connectorOrderBook := connector.OrderBook{
			Asset:     asset,
			Bids:      p.convertWSPriceLevels(wsUpdate.Bids),
			Asks:      p.convertWSPriceLevels(wsUpdate.Asks),
			Timestamp: wsUpdate.Timestamp,
		}

		select {
		case output <- connectorOrderBook:
		case <-p.wsContext.Done():
			return
		}
	}
}

func (p *Paradex) convertWSPriceLevels(wsLevels []websockets.PriceLevel) []connector.PriceLevel {
	result := make([]connector.PriceLevel, len(wsLevels))
	for i, wsLevel := range wsLevels {
		result[i] = connector.PriceLevel{
			Price:    wsLevel.Price,
			Quantity: wsLevel.Quantity,
		}
	}
	return result
}

func (p *Paradex) convertTradeUpdates(output chan<- connector.Trade) {
	defer close(output)

	for wsUpdate := range p.wsService.TradeUpdates() {
		side := padexmodel.ResponsesOrderSide(wsUpdate.Side)

		connectorTrade := connector.Trade{
			ID:        wsUpdate.TradeID,
			Symbol:    wsUpdate.Symbol,
			Price:     wsUpdate.Price,
			Quantity:  wsUpdate.Quantity,
			Side:      p.convertOrderSide(side),
			IsMaker:   false,        // Paradex doesn't provide this in trade updates
			Fee:       decimal.Zero, // Not available in WebSocket updates
			Timestamp: wsUpdate.Timestamp,
		}

		select {
		case output <- connectorTrade:
		case <-p.wsContext.Done():
			return
		}
	}
}

func (p *Paradex) parseAssetFromSymbol(symbol string) portfolio.Asset {
	// Parse symbols like "BTC-USD-PERP" to extract "BTC"
	parts := strings.Split(symbol, "-")
	if len(parts) > 0 {
		return portfolio.NewAsset(parts[0])
	}
	return portfolio.NewAsset(symbol)
}
