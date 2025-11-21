package paradex

import (
	"fmt"

	"github.com/backtesting-org/kronos-sdk/pkg/types/connector"
	"github.com/backtesting-org/kronos-sdk/pkg/types/portfolio"
	"github.com/shopspring/decimal"
)

func (p *Paradex) GetConnectorInfo() *connector.Info {
	return &connector.Info{
		Name:             connector.Paradex,
		TradingEnabled:   p.SupportsTradingOperations(),
		WebSocketEnabled: p.SupportsRealTimeData(),
		MaxLeverage:      decimal.NewFromFloat(10.0),
		SupportedOrderTypes: []connector.OrderType{
			connector.OrderTypeLimit,
			connector.OrderTypeMarket,
		},
		QuoteCurrency: "USD",
	}
}

func (p *Paradex) GetPerpSymbol(symbol portfolio.Asset) string {
	return fmt.Sprintf("%s-USD-PERP", symbol.Symbol())
}

func (p *Paradex) SupportsTradingOperations() bool {
	return true
}

func (p *Paradex) SupportsRealTimeData() bool {
	return true
}

func (p *Paradex) SupportsFundingRates() bool {
	return true
}

func (p *Paradex) SupportsPerpetuals() bool {
	return true
}

func (p *Paradex) SupportsSpot() bool {
	return false
}
